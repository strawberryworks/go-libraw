// Package apiinventory extracts and renders LibRaw public API inventory data.
package apiinventory

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SymbolKind identifies a LibRaw API symbol category.
type SymbolKind string

const (
	// FunctionKind is a public C API function declared in libraw.h.
	FunctionKind SymbolKind = "function"
	// EnumKind is a public enum declared in LibRaw headers.
	EnumKind SymbolKind = "enum"
	// MacroKind is a public LIBRAW_* preprocessor macro.
	MacroKind SymbolKind = "macro"
	// StructKind is a public libraw_* typedef struct type.
	StructKind SymbolKind = "struct"
	// VersionKind is a public version macro declared in libraw_version.h.
	VersionKind SymbolKind = "version"
)

// Symbol describes one upstream public LibRaw symbol.
type Symbol struct {
	Kind   SymbolKind
	Name   string
	Header string
}

// Inventory is a deterministic snapshot of selected LibRaw public headers.
type Inventory struct {
	HeaderDir string
	Version   string
	Symbols   []Symbol
}

// CoverageEntry records wrapper coverage status for a symbol.
type CoverageEntry struct {
	Kind   SymbolKind
	Name   string
	Status string
	Note   string
}

// Options configures inventory generation.
type Options struct {
	HeaderDir string
}

var (
	functionRE        = regexp.MustCompile(`(?m)^\s*DllDef\s+[^\n;]*\b(libraw_[A-Za-z0-9_]+)\s*\(`)
	enumRE            = regexp.MustCompile(`(?m)^\s*enum\s+([A-Za-z_][A-Za-z0-9_]*)`)
	macroRE           = regexp.MustCompile(`(?m)^\s*#\s*define\s+(LIBRAW_[A-Za-z0-9_]+)\b`)
	structRE          = regexp.MustCompile(`(?m)}\s*(libraw_[A-Za-z0-9_]+)\s*;`)
	versionRE         = regexp.MustCompile(`(?m)^\s*#\s*define\s+(LIBRAW_(?:MAJOR|MINOR|PATCH)_VERSION|LIBRAW_VERSION(?:_STR|_TAIL)?)\b(?:\s+(.+))?`)
	versionMacroNames = map[string]struct{}{
		"LIBRAW_MAJOR_VERSION": {},
		"LIBRAW_MINOR_VERSION": {},
		"LIBRAW_PATCH_VERSION": {},
		"LIBRAW_VERSION":       {},
		"LIBRAW_VERSION_STR":   {},
		"LIBRAW_VERSION_TAIL":  {},
	}
)

// FindHeaderDir returns the first directory that contains LibRaw public headers.
func FindHeaderDir(explicit string) (string, error) {
	candidates := []string{}
	if explicit != "" {
		candidates = append(candidates, explicit)
	}
	if env := os.Getenv("LIBRAW_HEADERS"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates,
		"/opt/homebrew/opt/libraw/include/libraw",
		"/usr/local/opt/libraw/include/libraw",
		"/usr/include/libraw",
		"/usr/local/include/libraw",
	)

	for _, candidate := range candidates {
		dir := normalizeHeaderDir(candidate)
		if hasRequiredHeaders(dir) {
			return dir, nil
		}
	}

	return "", errors.New("LibRaw headers not found; install LibRaw development files or set LIBRAW_HEADERS")
}

// Generate reads LibRaw public headers and returns their inventory.
func Generate(opts Options) (Inventory, error) {
	headerDir, err := FindHeaderDir(opts.HeaderDir)
	if err != nil {
		return Inventory{}, err
	}

	headerFiles := []string{
		"libraw.h",
		"libraw_const.h",
		"libraw_types.h",
		"libraw_version.h",
	}

	var symbols []Symbol
	versionParts := map[string]string{}
	seen := map[string]struct{}{}

	for _, header := range headerFiles {
		body, err := os.ReadFile(filepath.Join(headerDir, header))
		if err != nil {
			return Inventory{}, fmt.Errorf("read %s: %w", header, err)
		}
		clean := stripComments(string(body))

		addMatches(&symbols, seen, FunctionKind, header, functionRE.FindAllStringSubmatch(clean, -1))
		addMatches(&symbols, seen, EnumKind, header, enumRE.FindAllStringSubmatch(clean, -1))
		addMacroMatches(&symbols, seen, header, macroRE.FindAllStringSubmatch(clean, -1))
		addMatches(&symbols, seen, StructKind, header, structRE.FindAllStringSubmatch(clean, -1))

		for _, match := range versionRE.FindAllStringSubmatch(clean, -1) {
			name := match[1]
			addSymbol(&symbols, seen, VersionKind, name, header)
			if len(match) > 2 {
				versionParts[name] = strings.TrimSpace(match[2])
			}
		}
	}

	sortSymbols(symbols)

	return Inventory{
		HeaderDir: headerDir,
		Version:   versionString(versionParts),
		Symbols:   symbols,
	}, nil
}

// LoadCoverage reads a TSV coverage map.
func LoadCoverage(path string) (map[string]CoverageEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	entries := map[string]CoverageEntry{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			return nil, fmt.Errorf("%s:%d: expected 4 tab-separated fields", path, lineNo)
		}
		entry := CoverageEntry{
			Kind:   SymbolKind(parts[0]),
			Name:   parts[1],
			Status: parts[2],
			Note:   parts[3],
		}
		if entry.Kind == "" || entry.Name == "" || entry.Status == "" {
			return nil, fmt.Errorf("%s:%d: kind, name, and status are required", path, lineNo)
		}
		entries[symbolKey(entry.Kind, entry.Name)] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// RenderCoverageTSV renders a deterministic coverage map, preserving existing entries.
func RenderCoverageTSV(inv Inventory, coverage map[string]CoverageEntry) []byte {
	var out bytes.Buffer
	fmt.Fprintln(&out, "# kind\tname\tstatus\tnote")
	for _, sym := range inv.Symbols {
		entry, ok := coverage[symbolKey(sym.Kind, sym.Name)]
		if !ok {
			entry = CoverageEntry{
				Kind:   sym.Kind,
				Name:   sym.Name,
				Status: "deferred",
				Note:   "tracked for a future workflowr task",
			}
		}
		fmt.Fprintf(&out, "%s\t%s\t%s\t%s\n", entry.Kind, entry.Name, entry.Status, entry.Note)
	}
	return out.Bytes()
}

// RenderMarkdown renders an inventory and coverage map.
func RenderMarkdown(inv Inventory, coverage map[string]CoverageEntry) ([]byte, []Symbol) {
	var out bytes.Buffer
	missing := MissingCoverage(inv, coverage)

	fmt.Fprintln(&out, "# LibRaw API Inventory")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "- Header directory: `%s`\n", inv.HeaderDir)
	if inv.Version != "" {
		fmt.Fprintf(&out, "- Header version: `%s`\n", inv.Version)
	}
	fmt.Fprintf(&out, "- Total symbols: `%d`\n", len(inv.Symbols))
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "Statuses:")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "- `wrapped`: covered by the current Go API.")
	fmt.Fprintln(&out, "- `internal`: used behind the public Go API boundary.")
	fmt.Fprintln(&out, "- `deferred`: in scope for later workflowr tasks.")
	fmt.Fprintln(&out, "- `unsupported`: intentionally not planned.")
	fmt.Fprintln(&out, "- `unmapped`: present upstream but missing from coverage map.")
	fmt.Fprintln(&out)

	for _, kind := range []SymbolKind{FunctionKind, EnumKind, MacroKind, StructKind, VersionKind} {
		renderKind(&out, inv.Symbols, kind, coverage)
	}

	if len(missing) > 0 {
		fmt.Fprintln(&out, "## Missing Coverage")
		fmt.Fprintln(&out)
		for _, sym := range missing {
			fmt.Fprintf(&out, "- `%s` `%s` from `%s`\n", sym.Kind, sym.Name, sym.Header)
		}
		fmt.Fprintln(&out)
	}

	return out.Bytes(), missing
}

// MissingCoverage returns symbols without a coverage entry.
func MissingCoverage(inv Inventory, coverage map[string]CoverageEntry) []Symbol {
	missing := []Symbol{}
	for _, sym := range inv.Symbols {
		if _, ok := coverage[symbolKey(sym.Kind, sym.Name)]; !ok {
			missing = append(missing, sym)
		}
	}
	return missing
}

func renderKind(out *bytes.Buffer, symbols []Symbol, kind SymbolKind, coverage map[string]CoverageEntry) {
	fmt.Fprintf(out, "## %s\n\n", title(string(kind)))
	fmt.Fprintln(out, "| Symbol | Header | Status | Note |")
	fmt.Fprintln(out, "| --- | --- | --- | --- |")
	for _, sym := range symbols {
		if sym.Kind != kind {
			continue
		}
		entry, ok := coverage[symbolKey(sym.Kind, sym.Name)]
		status := "unmapped"
		note := "missing from coverage map"
		if ok {
			status = entry.Status
			note = entry.Note
		}
		fmt.Fprintf(out, "| `%s` | `%s` | `%s` | %s |\n", sym.Name, sym.Header, status, escapePipes(note))
	}
	fmt.Fprintln(out)
}

func normalizeHeaderDir(candidate string) string {
	if candidate == "" {
		return candidate
	}
	if hasRequiredHeaders(candidate) {
		return candidate
	}
	librawDir := filepath.Join(candidate, "libraw")
	if hasRequiredHeaders(librawDir) {
		return librawDir
	}
	includeDir := filepath.Join(candidate, "include", "libraw")
	if hasRequiredHeaders(includeDir) {
		return includeDir
	}
	return candidate
}

func hasRequiredHeaders(dir string) bool {
	if dir == "" {
		return false
	}
	for _, name := range []string{"libraw.h", "libraw_const.h", "libraw_types.h", "libraw_version.h"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

func addMatches(symbols *[]Symbol, seen map[string]struct{}, kind SymbolKind, header string, matches [][]string) {
	for _, match := range matches {
		if len(match) > 1 {
			addSymbol(symbols, seen, kind, match[1], header)
		}
	}
}

func addMacroMatches(symbols *[]Symbol, seen map[string]struct{}, header string, matches [][]string) {
	for _, match := range matches {
		if len(match) <= 1 {
			continue
		}
		name := match[1]
		if _, ok := versionMacroNames[name]; ok {
			continue
		}
		addSymbol(symbols, seen, MacroKind, name, header)
	}
}

func addSymbol(symbols *[]Symbol, seen map[string]struct{}, kind SymbolKind, name, header string) {
	key := symbolKey(kind, name)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*symbols = append(*symbols, Symbol{Kind: kind, Name: name, Header: header})
}

func sortSymbols(symbols []Symbol) {
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Kind != symbols[j].Kind {
			return symbols[i].Kind < symbols[j].Kind
		}
		return symbols[i].Name < symbols[j].Name
	})
}

func symbolKey(kind SymbolKind, name string) string {
	return string(kind) + "\t" + name
}

func stripComments(in string) string {
	blockRE := regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineRE := regexp.MustCompile(`(?m)//.*$`)
	return lineRE.ReplaceAllString(blockRE.ReplaceAllString(in, ""), "")
}

func versionString(parts map[string]string) string {
	major := parts["LIBRAW_MAJOR_VERSION"]
	minor := parts["LIBRAW_MINOR_VERSION"]
	patch := parts["LIBRAW_PATCH_VERSION"]
	if major == "" || minor == "" || patch == "" {
		return ""
	}
	tail := strings.Trim(parts["LIBRAW_VERSION_TAIL"], `"`)
	if tail == "" {
		return major + "." + minor + "." + patch
	}
	return major + "." + minor + "." + patch + "-" + tail
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:] + "s"
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
