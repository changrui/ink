package render

import (
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// msaa represents a multisample anti-aliased OpenGL texture.
// This is the main resource that shaders write to and read from.
//
// The renderer writes to a framebuffer containing a normal 2D texture.
// Then the renderer calls msaa.Paint() to blit that texture into an antialiased
// texture. Downstream shaders that read from this texture will read from the
// antialiased texture.
type msaa struct {
	ID   int
	Read struct {
		FBO, Tex uint32
	}
	Write struct {
		FBO, Tex uint32
	}
	Width, Height, Multisamples int
}

func newMsaa(id, w, h, multisamples int) msaa {

	m := msaa{
		ID:           id,
		Width:        w,
		Height:       h,
		Multisamples: multisamples,
	}

	// Create two textures:
	// 1. a multisampled texture which will be written to.
	// 2. a normal texture which will be read from.
	glGenTextures(1, &m.Read.Tex)
	glGenTextures(1, &m.Write.Tex)

	// ...and two framebuffers, one for each texture.
	glGenFramebuffers(1, &m.Read.FBO)
	glGenFramebuffers(1, &m.Write.FBO)

	// Initialize the Read texture
	glBindFramebuffer(gl.FRAMEBUFFER, m.Read.FBO)

	glBindTexture(gl.TEXTURE_2D, m.Read.Tex)
	glTexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(m.Width),
		int32(m.Height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil)

	glTexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glTexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	glTexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	glTexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	glFramebufferTexture2D(
		gl.FRAMEBUFFER,
		gl.COLOR_ATTACHMENT0,
		gl.TEXTURE_2D,
		m.Read.Tex,
		0,
	)

	// Initialize the Write texture
	glBindFramebuffer(gl.FRAMEBUFFER, m.Write.FBO)

	glBindTexture(gl.TEXTURE_2D_MULTISAMPLE, m.Write.Tex)
	glTexImage2DMultisample(
		gl.TEXTURE_2D_MULTISAMPLE,
		int32(m.Multisamples),
		gl.RGBA,
		int32(m.Width),
		int32(m.Height),
		false,
	)

	glFramebufferTexture2D(
		gl.FRAMEBUFFER,
		gl.COLOR_ATTACHMENT0,
		gl.TEXTURE_2D_MULTISAMPLE,
		m.Write.Tex,
		0,
	)

	m.Clear()
	return m
}

func (m *msaa) Clear() {
	glBindFramebuffer(gl.FRAMEBUFFER, m.Read.FBO)
	glClearColor(0, 0, 0, 1)
	glClear(gl.COLOR_BUFFER_BIT)

	glBindFramebuffer(gl.FRAMEBUFFER, m.Write.FBO)
	glClearColor(0, 0, 0, 1)
	glClear(gl.COLOR_BUFFER_BIT)
}

// Blit the "write" texture into the anti-aliased "read" texture.
func (m *msaa) Paint() {
	// Copy the multisample texture (Write) to the regular texture (Read).
	glBindFramebuffer(gl.DRAW_FRAMEBUFFER, m.Read.FBO)
	glBindFramebuffer(gl.READ_FRAMEBUFFER, m.Write.FBO)

	glBlitFramebuffer(
		0, 0, int32(m.Width), int32(m.Height),
		0, 0, int32(m.Width), int32(m.Height),
		gl.COLOR_BUFFER_BIT,
		gl.LINEAR,
	)
}

// Blit the anti-aliased "read" texture to the given framebuffer ID.
// Used during compisiting to copy textures to the screen.
func (m *msaa) Blit(to uint32) {
	glBindFramebuffer(gl.DRAW_FRAMEBUFFER, to)
	glBindFramebuffer(gl.READ_FRAMEBUFFER, m.Read.FBO)

	glBlitFramebuffer(
		0, 0, int32(m.Width), int32(m.Height),
		0, 0, int32(m.Width), int32(m.Height),
		gl.COLOR_BUFFER_BIT,
		gl.LINEAR,
	)
}

func (m *msaa) Pixels(x, y, w, h float32) []uint8 {

	glBindFramebuffer(gl.READ_FRAMEBUFFER, m.Read.FBO)
	xi := int(x * float32(m.Width))
	yi := int(y * float32(m.Height))
	wi := int(w * float32(m.Width))
	hi := int(h * float32(m.Height))

	if xi < 0 {
		xi = 0
	}
	if yi < 0 {
		yi = 0
	}
	if wi > m.Width {
		wi = m.Width
	}
	if hi > m.Height {
		hi = m.Height
	}

	pixels := make([]uint8, wi*hi*4)

	// TODO how to allow flexible querying without complex API?
	//      e.g. get only red pixels
	glReadPixels(
		int32(xi),
		int32(yi),
		int32(wi),
		int32(hi),
		gl.RGBA, gl.UNSIGNED_BYTE, glPtr(pixels),
	)

	return pixels
}

func (m *msaa) Image() image.Image {

	glBindFramebuffer(gl.READ_FRAMEBUFFER, m.Read.FBO)

	pixels := make([]uint8, m.Width*m.Height*4)
	glReadPixels(
		0, 0,
		int32(m.Width), int32(m.Height),
		gl.RGBA, gl.UNSIGNED_BYTE, glPtr(pixels),
	)

	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			// the orientation of PNG vs OpenGL is upside-down.
			i := ((m.Height - y - 1) * m.Width * 4) + (x * 4)

			img.SetRGBA(x, y, color.RGBA{
				R: pixels[i+0],
				G: pixels[i+1],
				B: pixels[i+2],
				// alpha is premultiplied at this point.
				// TODO difficult to retrieve pixel data where alpha hasn't been premultiplied
				A: 255,
			})
		}
	}

	return img
}

func (m *msaa) Destroy() {
	if m == nil {
		return
	}
	glDeleteTextures(1, &m.Read.Tex)
	glDeleteTextures(1, &m.Write.Tex)
	glDeleteFramebuffers(1, &m.Read.FBO)
	glDeleteFramebuffers(1, &m.Write.FBO)
}
