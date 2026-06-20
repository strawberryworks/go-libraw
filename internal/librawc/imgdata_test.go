//go:build cgo

package librawc

import (
	"runtime"
	"slices"
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

func TestHandleDirectRawBuffersCopyRows(t *testing.T) {
	h, err := New(0)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer h.Close()

	const (
		width    = 2
		height   = 2
		rowPitch = 40
	)
	h.ptr.sizes.raw_width = width
	h.ptr.sizes.raw_height = height
	h.ptr.sizes.raw_pitch = rowPitch

	color3 := []uint16{
		1, 2, 3, 4, 5, 6, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104,
		7, 8, 9, 10, 11, 12, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204,
	}
	color4 := []uint16{
		21, 22, 23, 24, 25, 26, 27, 28, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102,
		29, 30, 31, 32, 33, 34, 35, 36, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202,
	}
	floatImage := []float32{
		1.5, 2.5, 91, 92, 93, 94, 95, 96, 97, 98,
		3.5, 4.5, 191, 192, 193, 194, 195, 196, 197, 198,
	}
	float3 := []float32{
		10, 11, 12, 13, 14, 15, 91, 92, 93, 94,
		16, 17, 18, 19, 20, 21, 95, 96, 97, 98,
	}
	float4 := []float32{
		30, 31, 32, 33, 34, 35, 36, 37, 91, 92,
		38, 39, 40, 41, 42, 43, 44, 45, 95, 96,
	}

	*(*unsafe.Pointer)(unsafe.Pointer(&h.ptr.rawdata.color3_image)) = unsafe.Pointer(&color3[0])
	*(*unsafe.Pointer)(unsafe.Pointer(&h.ptr.rawdata.color4_image)) = unsafe.Pointer(&color4[0])
	*(*unsafe.Pointer)(unsafe.Pointer(&h.ptr.rawdata.float_image)) = unsafe.Pointer(&floatImage[0])
	*(*unsafe.Pointer)(unsafe.Pointer(&h.ptr.rawdata.float3_image)) = unsafe.Pointer(&float3[0])
	*(*unsafe.Pointer)(unsafe.Pointer(&h.ptr.rawdata.float4_image)) = unsafe.Pointer(&float4[0])

	gotColor3 := h.Color3Image()
	if want := [][3]uint16{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12}}; !slices.Equal(gotColor3, want) {
		t.Fatalf("Color3Image() = %v, want %v", gotColor3, want)
	}

	gotColor4 := h.Color4Image()
	if want := [][4]uint16{{21, 22, 23, 24}, {25, 26, 27, 28}, {29, 30, 31, 32}, {33, 34, 35, 36}}; !slices.Equal(gotColor4, want) {
		t.Fatalf("Color4Image() = %v, want %v", gotColor4, want)
	}

	gotFloatImage := h.FloatImage()
	if want := []float32{1.5, 2.5, 3.5, 4.5}; !slices.Equal(gotFloatImage, want) {
		t.Fatalf("FloatImage() = %v, want %v", gotFloatImage, want)
	}

	gotFloat3 := h.Float3Image()
	if want := [][3]float32{{10, 11, 12}, {13, 14, 15}, {16, 17, 18}, {19, 20, 21}}; !slices.Equal(gotFloat3, want) {
		t.Fatalf("Float3Image() = %v, want %v", gotFloat3, want)
	}

	gotFloat4 := h.Float4Image()
	if want := [][4]float32{{30, 31, 32, 33}, {34, 35, 36, 37}, {38, 39, 40, 41}, {42, 43, 44, 45}}; !slices.Equal(gotFloat4, want) {
		t.Fatalf("Float4Image() = %v, want %v", gotFloat4, want)
	}

	copy(color3, []uint16{99, 99, 99})
	if gotColor3[0] != [3]uint16{1, 2, 3} {
		t.Fatalf("Color3Image() result aliases C memory: got[0] = %v", gotColor3[0])
	}
	runtime.KeepAlive(color3)
	runtime.KeepAlive(color4)
	runtime.KeepAlive(floatImage)
	runtime.KeepAlive(float3)
	runtime.KeepAlive(float4)
}

func TestHandleRawRowPitchFallsBackToDensePitch(t *testing.T) {
	h, err := New(0)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer h.Close()

	h.ptr.sizes.raw_width = 3
	h.ptr.sizes.raw_height = 4
	h.ptr.sizes.raw_pitch = 0

	width, height := h.rawDimensions()
	if width != 3 || height != 4 {
		t.Fatalf("rawDimensions() = %d, %d; want 3, 4", width, height)
	}
	if got := h.rawRowPitch(4, 2); got != 24 {
		t.Fatalf("rawRowPitch() = %d, want dense pitch 24", got)
	}
}
