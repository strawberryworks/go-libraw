//go:build cgo

package librawc

/*
#include <stdlib.h>
#include <libraw/libraw.h>
*/
import "C"

import "unsafe"

const maxInt = int(^uint(0) >> 1)

// Color returns LibRaw's color index for the sensor pixel at (row, col),
// mapping the camera's color filter array. It wraps libraw_COLOR.
func (h *Handle) Color(row, col int) int {
	return int(C.libraw_COLOR(h.ptr, C.int(row), C.int(col)))
}

// RawWidth returns the raw image width in pixels (libraw_get_raw_width).
func (h *Handle) RawWidth() int {
	return int(C.libraw_get_raw_width(h.ptr))
}

// RawHeight returns the raw image height in pixels (libraw_get_raw_height).
func (h *Handle) RawHeight() int {
	return int(C.libraw_get_raw_height(h.ptr))
}

// RawImage returns a Go copy of the single-channel raw Bayer buffer
// (imgdata.rawdata.raw_image), or nil if no such buffer is available.
//
// The buffer is row-padded: its length is (raw_pitch/2)*raw_height samples.
func (h *Handle) RawImage() []uint16 {
	img := h.ptr.rawdata.raw_image
	if img == nil {
		return nil
	}
	height := int(h.ptr.sizes.raw_height)
	pitch := int(h.ptr.sizes.raw_pitch)
	if pitch == 0 {
		pitch = int(h.ptr.sizes.raw_width) * 2
	}
	n := (pitch / 2) * height
	if n <= 0 {
		return nil
	}
	src := unsafe.Slice((*uint16)(unsafe.Pointer(img)), n)
	out := make([]uint16, n)
	copy(out, src)
	return out
}

// Color3Image returns a Go copy of imgdata.rawdata.color3_image, or nil if no
// such buffer is available. The returned slice contains raw_width*raw_height
// RGB pixels and skips any row padding described by raw_pitch.
func (h *Handle) Color3Image() [][3]uint16 {
	img := h.ptr.rawdata.color3_image
	if img == nil {
		return nil
	}
	width, height := h.rawDimensions()
	return copyPixelRows[[3]uint16](unsafe.Pointer(img), width, height, h.rawRowPitch(3, 2))
}

// Color4Image returns a Go copy of imgdata.rawdata.color4_image, or nil if no
// such buffer is available. The returned slice contains raw_width*raw_height
// RGBA pixels and skips any row padding described by raw_pitch.
func (h *Handle) Color4Image() [][4]uint16 {
	img := h.ptr.rawdata.color4_image
	if img == nil {
		return nil
	}
	width, height := h.rawDimensions()
	return copyPixelRows[[4]uint16](unsafe.Pointer(img), width, height, h.rawRowPitch(4, 2))
}

// Float3Image returns a Go copy of imgdata.rawdata.float3_image, or nil if no
// such buffer is available. The returned slice contains raw_width*raw_height
// RGB float pixels and skips any row padding described by raw_pitch.
func (h *Handle) Float3Image() [][3]float32 {
	img := h.ptr.rawdata.float3_image
	if img == nil {
		return nil
	}
	width, height := h.rawDimensions()
	return copyPixelRows[[3]float32](unsafe.Pointer(img), width, height, h.rawRowPitch(3, 4))
}

// Float4Image returns a Go copy of imgdata.rawdata.float4_image, or nil if no
// such buffer is available. The returned slice contains raw_width*raw_height
// RGBA float pixels and skips any row padding described by raw_pitch.
func (h *Handle) Float4Image() [][4]float32 {
	img := h.ptr.rawdata.float4_image
	if img == nil {
		return nil
	}
	width, height := h.rawDimensions()
	return copyPixelRows[[4]float32](unsafe.Pointer(img), width, height, h.rawRowPitch(4, 4))
}

// copyPixelRows copies width*height elements of type T from the row-padded C
// buffer at base into a Go-owned slice, skipping the padding described by
// rowPitch (in bytes). T must have the same memory layout as the C element
// type (C.ushort is uint16, C.float is float32, and fixed C arrays match Go
// arrays), so the copy is a direct reinterpretation. It returns nil for
// non-positive or overflowing geometry.
func copyPixelRows[T any](base unsafe.Pointer, width, height, rowPitch int) []T {
	if !validRawBufferGeometry(width, height, rowPitch) {
		return nil
	}
	out := make([]T, width*height)
	for row := 0; row < height; row++ {
		src := unsafe.Slice((*T)(unsafe.Add(base, row*rowPitch)), width)
		copy(out[row*width:(row+1)*width], src)
	}
	return out
}

// FloatImage returns a Go copy of imgdata.rawdata.float_image, or nil if no
// such buffer is available. The returned slice contains raw_width*raw_height
// single-channel float samples and skips any row padding described by raw_pitch.
func (h *Handle) FloatImage() []float32 {
	img := h.ptr.rawdata.float_image
	if img == nil {
		return nil
	}
	width, height := h.rawDimensions()
	return copyPixelRows[float32](unsafe.Pointer(img), width, height, h.rawRowPitch(1, 4))
}

func (h *Handle) rawDimensions() (int, int) {
	return int(h.ptr.sizes.raw_width), int(h.ptr.sizes.raw_height)
}

func (h *Handle) rawRowPitch(channels, bytesPerChannel int) int {
	width := int(h.ptr.sizes.raw_width)
	pitch := int(h.ptr.sizes.raw_pitch)
	denseBytes := width * channels * bytesPerChannel
	if pitch >= denseBytes {
		return pitch
	}
	return denseBytes
}

func validRawBufferGeometry(width, height, rowPitch int) bool {
	if width <= 0 || height <= 0 || rowPitch <= 0 {
		return false
	}
	if height > maxInt/rowPitch {
		return false
	}
	if height > maxInt/width {
		return false
	}
	return true
}

// FourChannels returns a Go copy of the 4-channel postprocessing image buffer
// (imgdata.image) as a flat slice of [4]uint16 pixels, or nil when the buffer
// is not available (Raw2Image or DcrawProcess must have run first).
//
// The length is iheight*iwidth. Channel assignment follows the CFA pattern
// (typically RGBG for Bayer sensors).
func (h *Handle) FourChannels() [][4]uint16 {
	if h.ptr.image == nil {
		return nil
	}
	n := int(h.ptr.sizes.iheight) * int(h.ptr.sizes.iwidth)
	if n <= 0 {
		return nil
	}
	src := unsafe.Slice(h.ptr.image, n)
	out := make([][4]uint16, n)
	for i, px := range src {
		out[i] = [4]uint16{uint16(px[0]), uint16(px[1]), uint16(px[2]), uint16(px[3])}
	}
	return out
}

// ThumbnailData returns a Go copy of the unpacked thumbnail bytes
// (imgdata.thumbnail.thumb), or nil if no thumbnail data is present.
func (h *Handle) ThumbnailData() []byte {
	thumb := h.ptr.thumbnail.thumb
	n := int(h.ptr.thumbnail.tlength)
	if thumb == nil || n <= 0 {
		return nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(thumb)), n)
	out := make([]byte, n)
	copy(out, src)
	return out
}
