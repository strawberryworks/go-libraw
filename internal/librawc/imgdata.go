//go:build cgo

package librawc

/*
#include <stdlib.h>
#include <libraw/libraw.h>
*/
import "C"

import "unsafe"

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

// ThumbnailData returns a Go copy of the unpacked thumbnail bytes
// (imgdata.thumbnail.thumb), or nil if no thumbnail data is present.
func (h *Handle) ThumbnailData() []byte {
	thumb := h.ptr.thumbnail.thumb
	n := int(h.ptr.thumbnail.tlength)
	if thumb == nil || n <= 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(thumb), C.int(n))
}
