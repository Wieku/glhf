package glhf

import (
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Wrap int32

const (
	CLAMP_TO_EDGE = Wrap(gl.CLAMP_TO_EDGE)
	CLAMP_TO_BORDER = Wrap(gl.CLAMP_TO_BORDER)
	REPEAT = Wrap(gl.REPEAT)
)

// Texture is an OpenGL texture.
type Texture struct {
	tex           binder
	width, height, mipmapLevels int
	smooth        bool
}

// NewTexture creates a new texture with the specified width and height with some initial
// pixel values. The pixels must be a sequence of RGBA values (one byte per component).
func NewTexture(width, height, mipmapLevels int, smooth bool, pixels []uint8) *Texture {
	tex := &Texture{
		tex: binder{
			restoreLoc: gl.TEXTURE_BINDING_2D,
			bindFunc: func(obj uint32) {
				gl.BindTexture(gl.TEXTURE_2D, obj)
			},
		},
		width:  width,
		height: height,
		mipmapLevels: mipmapLevels,
	}

	gl.GenTextures(1, &tex.tex.obj)

	tex.Begin()
	defer tex.End()

	if mipmapLevels > 0 {
		gl.TexStorage2D(gl.TEXTURE_2D, int32(mipmapLevels), gl.RGBA8, int32(width), int32(height))
		gl.TexSubImage2D(
			gl.TEXTURE_2D,
			0,
			0,
			0,
			int32(width),
			int32(height),
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(pixels),
		)
		gl.GenerateMipmap(gl.TEXTURE_2D)
	} else {
		gl.TexImage2D(
			gl.TEXTURE_2D,
			0,
			gl.RGBA,
			int32(width),
			int32(height),
			0,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(pixels),
		)
	}

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

	tex.SetSmooth(smooth)
	tex.SetWrap(CLAMP_TO_BORDER)

	runtime.SetFinalizer(tex, (*Texture).delete)

	return tex
}

func (t *Texture) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteTextures(1, &t.tex.obj)
	})
}

// ID returns the OpenGL ID of this Texture.
func (t *Texture) ID() uint32 {
	return t.tex.obj
}

// Width returns the width of the Texture in pixels.
func (t *Texture) Width() int {
	return t.width
}

// Height returns the height of the Texture in pixels.
func (t *Texture) Height() int {
	return t.height
}

// SetPixels sets the content of a sub-region of the Texture. Pixels must be an RGBA byte sequence.
func (t *Texture) SetPixels(x, y, w, h int, pixels []uint8) {
	if len(pixels) != w*h*4 {
		panic("set pixels: wrong number of pixels")
	}
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		int32(x),
		int32(y),
		int32(w),
		int32(h),
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(pixels),
	)

	if t.mipmapLevels > 0 {
		gl.GenerateMipmap(gl.TEXTURE_2D)
	}

}

// Pixels returns the content of a sub-region of the Texture as an RGBA byte sequence.
func (t *Texture) Pixels(x, y, w, h int) []uint8 {
	pixels := make([]uint8, t.width*t.height*4)
	gl.GetTexImage(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(pixels),
	)
	subPixels := make([]uint8, w*h*4)
	for i := 0; i < h; i++ {
		row := pixels[(i+y)*t.width*4+x*4 : (i+y)*t.width*4+(x+w)*4]
		subRow := subPixels[i*w*4 : (i+1)*w*4]
		copy(subRow, row)
	}
	return subPixels
}

// SetSmooth sets whether the Texture should be drawn "smoothly" or "pixely".
//
// It affects how the Texture is drawn when zoomed. Smooth interpolates between the neighbour
// pixels, while pixely always chooses the nearest pixel.
func (t *Texture) SetSmooth(smooth bool) {
	t.smooth = smooth
	if smooth {
		if t.mipmapLevels > 0 {
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		} else {
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		}

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	} else {
		if t.mipmapLevels > 0 {
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
		} else {
			gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		}

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	}
}

func (t *Texture) SetWrap(wrap Wrap) {
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, int32(wrap))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, int32(wrap))
}

// Smooth returns whether the Texture is set to be drawn "smooth" or "pixely".
func (t *Texture) Smooth() bool {
	return t.smooth
}

// Begin binds the Texture. This is necessary before using the Texture.
func (t *Texture) Begin() {
	t.tex.bind()
}

// End unbinds the Texture and restores the previous one.
func (t *Texture) End() {
	t.tex.restore()
}
