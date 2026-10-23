package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Glyph kinds for the tray's file rows: the web page's line icons (24
// unit grid), drawn in their state colour on a transparent ground.
const (
	GlyphUp       = "up"
	GlyphDown     = "down"
	GlyphCheck    = "check"
	GlyphConflict = "conflict"
	GlyphFailed   = "failed"
	GlyphDeleted  = "deleted"
)

type stroke [][]pt // polylines

var glyphs = map[string]struct {
	lines stroke
	dots  []pt
	col   color.RGBA
}{
	GlyphUp:       {stroke{{{12, 19}, {12, 5}}, {{6.5, 10.5}, {12, 5}, {17.5, 10.5}}}, nil, rgb(0x0061fe)},
	GlyphDown:     {stroke{{{12, 5}, {12, 19}}, {{6.5, 13.5}, {12, 19}, {17.5, 13.5}}}, nil, rgb(0x0061fe)},
	GlyphCheck:    {stroke{{{5, 12.5}, {9.5, 17}, {19, 7.5}}}, nil, rgb(0x1f7a4d)},
	GlyphConflict: {stroke{{{12, 4}, {21, 19.5}, {3, 19.5}, {12, 4}}, {{12, 9.5}, {12, 14}}}, []pt{{12, 16.8}}, rgb(0xa85d00)},
	GlyphFailed:   {stroke{{{6.5, 6.5}, {17.5, 17.5}}, {{17.5, 6.5}, {6.5, 17.5}}}, nil, rgb(0xc4312b)},
	GlyphDeleted:  {stroke{{{6.5, 6.5}, {17.5, 17.5}}, {{17.5, 6.5}, {6.5, 17.5}}}, nil, rgb(0x8f877f)},
}

// DrawGlyph is a glyph at n×n pixels, 4×4 supersampled.
func DrawGlyph(kind string, n int) *image.RGBA {
	g := glyphs[kind]
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	u := 24 / float64(n)
	half := math.Max(1.1, 0.9*u) // stroke ≈ 2.2 units, never under ~1.8px
	const ss = 4
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			hit := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					p := pt{(float64(x) + (float64(sx)+.5)/ss) * u, (float64(y) + (float64(sy)+.5)/ss) * u}
					on := false
					for _, l := range g.lines {
						for i := 0; i+1 < len(l) && !on; i++ {
							on = segDist(p, l[i], l[i+1]) <= half
						}
					}
					for _, d := range g.dots {
						on = on || within(p, d, half*1.15)
					}
					if on {
						hit++
					}
				}
			}
			if hit == 0 {
				continue
			}
			a := float64(hit) / ss / ss
			img.SetRGBA(x, y, color.RGBA{uint8(float64(g.col.R) * a), uint8(float64(g.col.G) * a), uint8(float64(g.col.B) * a), uint8(255 * a)})
		}
	}
	return img
}

// GlyphPNG is a glyph as a PNG.
func GlyphPNG(kind string, n int) []byte {
	var b bytes.Buffer
	png.Encode(&b, DrawGlyph(kind, n))
	return b.Bytes()
}
