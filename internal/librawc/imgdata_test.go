//go:build cgo

package librawc

import (
	"testing"
	"unsafe"
)

func TestValidRawBufferGeometry(t *testing.T) {
	if !validRawBufferGeometry(2, 2, 14) {
		t.Fatal("validRawBufferGeometry() = false, want true")
	}
}

func TestValidRawBufferGeometryRejectsInvalidGeometry(t *testing.T) {
	tests := []struct {
		name                 string
		width, height, pitch int
	}{
		{name: "zero width", width: 0, height: 2, pitch: 12},
		{name: "zero height", width: 2, height: 0, pitch: 12},
		{name: "zero pitch", width: 2, height: 2, pitch: 0},
		{name: "pitch overflow", width: 2, height: maxInt, pitch: 2},
		{name: "area overflow", width: maxInt, height: maxInt, pitch: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if validRawBufferGeometry(tt.width, tt.height, tt.pitch) {
				t.Fatal("validRawBufferGeometry() = true, want false")
			}
		})
	}
}

// TestCopyPixelRowsDense copies a tightly packed buffer with no row padding.
func TestCopyPixelRowsDense(t *testing.T) {
	backing := [][3]uint16{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12}}
	const rowPitch = 2 * 3 * 2 // 2 pixels * 3 channels * 2 bytes

	got := copyPixelRows[[3]uint16](unsafe.Pointer(&backing[0]), 2, 2, rowPitch)
	if len(got) != 4 {
		t.Fatalf("copyPixelRows len = %d, want 4", len(got))
	}
	if got[0] != [3]uint16{1, 2, 3} || got[3] != [3]uint16{10, 11, 12} {
		t.Fatalf("copyPixelRows = %v, want first {1 2 3} last {10 11 12}", got)
	}

	// The result is a Go-owned copy: mutating the source must not change it.
	backing[0] = [3]uint16{99, 99, 99}
	if got[0] != [3]uint16{1, 2, 3} {
		t.Fatalf("copyPixelRows result aliases source: got[0] = %v", got[0])
	}
}

// TestCopyPixelRowsSkipsPadding verifies the per-row padding implied by a
// rowPitch wider than the dense width is skipped.
func TestCopyPixelRowsSkipsPadding(t *testing.T) {
	// Two rows of 8 uint16 each (16 bytes); only the first 6 hold pixel data,
	// the last 2 are padding.
	backing := []uint16{
		1, 2, 3, 4, 5, 6, 0, 0,
		7, 8, 9, 10, 11, 12, 0, 0,
	}
	const rowPitch = 8 * 2 // 16 bytes per row

	got := copyPixelRows[[3]uint16](unsafe.Pointer(&backing[0]), 2, 2, rowPitch)
	if len(got) != 4 {
		t.Fatalf("copyPixelRows len = %d, want 4", len(got))
	}
	want := [][3]uint16{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12}}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("copyPixelRows[%d] = %v, want %v", i, got[i], w)
		}
	}
}

func TestCopyPixelRowsRejectsInvalidGeometry(t *testing.T) {
	backing := [][3]uint16{{1, 2, 3}}
	if got := copyPixelRows[[3]uint16](unsafe.Pointer(&backing[0]), 0, 2, 12); got != nil {
		t.Fatalf("copyPixelRows with zero width = %v, want nil", got)
	}
}

// TestCopyPixelRowsFloat exercises the float element path.
func TestCopyPixelRowsFloat(t *testing.T) {
	backing := []float32{1, 2, 3, 4}
	const rowPitch = 2 * 4 // 2 single-channel float32 samples per row

	got := copyPixelRows[float32](unsafe.Pointer(&backing[0]), 2, 2, rowPitch)
	if len(got) != 4 {
		t.Fatalf("copyPixelRows len = %d, want 4", len(got))
	}
	if got[0] != 1 || got[3] != 4 {
		t.Fatalf("copyPixelRows = %v, want first 1 last 4", got)
	}
}
