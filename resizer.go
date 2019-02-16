package cropper

import (
	"image"

	"github.com/nfnt/resize"
)

// Resizer is used to resize images. See the nfnt package for a default implementation using
// github.com/nfnt/resize.
type Resizer interface {
	Resize(img image.Image, width, height uint) image.Image
}

type nfntResizer struct {
	interpolationType resize.InterpolationFunction
}

func (r nfntResizer) Resize(img image.Image, width, height uint) image.Image {
	return resize.Resize(width, height, img, r.interpolationType)
}

// NewResizer creates a new Resizer with the given interpolation type.
func NewResizer(interpolationType resize.InterpolationFunction) Resizer {
	return nfntResizer{interpolationType: interpolationType}
}

// NewDefaultResizer creates a new Resizer with the default interpolation type.
func NewDefaultResizer() Resizer {
	return NewResizer(resize.Bicubic)
}
