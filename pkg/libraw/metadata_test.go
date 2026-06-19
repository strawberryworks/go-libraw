//go:build cgo

package libraw

import (
	"errors"
	"path/filepath"
	"testing"
)

var metadataFixtures = []string{
	"../../testdata/RAW_CANON_R6.CR3",
	"../../testdata/RAW_NIKON_ZFC.NEF",
	"../../testdata/RAW_RICOH_GR3X.DNG",
	"../../testdata/RAW_SONY_ILCE-7M4.ARW",
}

func TestMetadataForFixtures(t *testing.T) {
	for _, fixture := range metadataFixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			p := openProcessor(t)
			if err := p.OpenFile(fixture); err != nil {
				t.Fatalf("OpenFile(%q) error = %v", fixture, err)
			}
			meta, err := p.Metadata()
			if err != nil {
				t.Fatalf("Metadata() error = %v", err)
			}

			if meta.ID.Make == "" {
				t.Fatalf("ID.Make is empty: %+v", meta.ID)
			}
			if meta.ID.Model == "" {
				t.Fatalf("ID.Model is empty: %+v", meta.ID)
			}
			if meta.Sizes.RawWidth == 0 || meta.Sizes.RawHeight == 0 {
				t.Fatalf("raw dimensions = %dx%d, want non-zero", meta.Sizes.RawWidth, meta.Sizes.RawHeight)
			}
			if meta.Sizes.Width == 0 || meta.Sizes.Height == 0 {
				t.Fatalf("image dimensions = %dx%d, want non-zero", meta.Sizes.Width, meta.Sizes.Height)
			}
			if meta.ID.Colors <= 0 {
				t.Fatalf("ID.Colors = %d, want positive", meta.ID.Colors)
			}
			if meta.Color.Maximum == 0 && meta.Color.DataMaximum == 0 {
				t.Fatalf("color maximums are empty: maximum=%d data_maximum=%d", meta.Color.Maximum, meta.Color.DataMaximum)
			}
			if meta.Other.ISOSpeed == 0 {
				t.Fatalf("Other.ISOSpeed = %v, want non-zero", meta.Other.ISOSpeed)
			}
		})
	}
}

func TestMetadataSnapshotSurvivesClose(t *testing.T) {
	p, err := NewProcessor()
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	if err := p.OpenFile(sampleRAW); err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	meta, err := p.Metadata()
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if meta.ID.Make == "" || meta.Sizes.RawWidth == 0 {
		t.Fatalf("metadata snapshot was not populated before close: %+v", meta)
	}
}

func TestMetadataRawDataSummaryBuffersAndCBlackFull(t *testing.T) {
	p := openProcessor(t)
	if err := p.OpenFile(sampleRAW); err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if err := p.Unpack(); err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}

	meta, err := p.Metadata()
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if !meta.RawData.HasRawImage {
		t.Fatal("RawData.HasRawImage = false, want true after Unpack for sample fixture")
	}
	if meta.RawData.HasColor3Image || meta.RawData.HasColor4Image || meta.RawData.HasFloat3Image || meta.RawData.HasFloat4Image {
		t.Logf("fixture exposes direct raw buffers: rawdata=%+v", meta.RawData)
	}
	if got := len(meta.Color.CBlackFull); got <= len(meta.Color.CBlack) {
		t.Fatalf("Color.CBlackFull len = %d, want more than %d", got, len(meta.Color.CBlack))
	}
	if got := len(meta.RawData.Color.CBlackFull); got != len(meta.Color.CBlackFull) {
		t.Fatalf("RawData.Color.CBlackFull len = %d, want %d", got, len(meta.Color.CBlackFull))
	}
	for i, want := range meta.Color.CBlack {
		if got := meta.Color.CBlackFull[i]; got != want {
			t.Fatalf("Color.CBlackFull[%d] = %d, want first CBlack value %d", i, got, want)
		}
		if got := meta.RawData.Color.CBlackFull[i]; got != meta.RawData.Color.CBlack[i] {
			t.Fatalf("RawData.Color.CBlackFull[%d] = %d, want first RawData.Color.CBlack value %d", i, got, meta.RawData.Color.CBlack[i])
		}
	}

	dng := meta.Color.DNGLevels
	if got := len(dng.DNGCBlackFull); got <= len(dng.DNGCBlack) {
		t.Fatalf("DNGLevels.DNGCBlackFull len = %d, want more than %d", got, len(dng.DNGCBlack))
	}
	if got := len(dng.DNGFCBlackFull); got != len(dng.DNGCBlackFull) {
		t.Fatalf("DNGLevels.DNGFCBlackFull len = %d, want %d", got, len(dng.DNGCBlackFull))
	}
	for i, want := range dng.DNGCBlack {
		if got := dng.DNGCBlackFull[i]; got != want {
			t.Fatalf("DNGLevels.DNGCBlackFull[%d] = %d, want first DNGCBlack value %d", i, got, want)
		}
	}
	for i, want := range dng.DNGFCBlack {
		if got := dng.DNGFCBlackFull[i]; got != want {
			t.Fatalf("DNGLevels.DNGFCBlackFull[%d] = %v, want first DNGFCBlack value %v", i, got, want)
		}
	}
}

func TestMetadataAfterCloseReturnsErrClosed(t *testing.T) {
	p, err := NewProcessor()
	if err != nil {
		t.Fatalf("NewProcessor() error = %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := p.Metadata(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Metadata() after Close error = %v, want ErrClosed", err)
	}
}

func TestMetadataThumbnailList(t *testing.T) {
	p := openProcessor(t)
	if err := p.OpenFile(sampleThumbRAW); err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	meta, err := p.Metadata()
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}
	if meta.Thumbs.Count == 0 {
		t.Fatal("Thumbs.Count = 0, want at least one thumbnail entry")
	}
	if len(meta.Thumbs.Items) == 0 {
		t.Fatal("Thumbs.Items is empty")
	}
	// LibRaw's thumbnail list can include preview entries it does not assign
	// dimensions to (CR3, for example, exposes multiple preview types). Require
	// at least one sized thumbnail rather than every entry being non-zero.
	sized := 0
	for _, item := range meta.Thumbs.Items {
		if item.Width > 0 && item.Height > 0 {
			sized++
		}
	}
	if sized == 0 {
		t.Fatalf("no thumbnail entry has non-zero dimensions: %+v", meta.Thumbs.Items)
	}
}
