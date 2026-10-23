// Package icon draws Drive's icon, so there are no binary assets. It is
// desk/web/tile.svg, rasterized here: a rounded square with a diagonal
// gradient, a white folder, and two circular sync arrows. A Style per
// state colours it. The tray uses it at 16–32px (with thicker arrows, so
// they stay legible); packaging renders the app icons from it
// (scripts/genicon).
package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Style is a state's colours: the background gradient (top-left to
// bottom-right) and the arrows.
type Style struct{ Bg0, Bg1, Arrow color.RGBA }

var (
	Idle = Style{rgb(0x3d8bfd), rgb(0x1f4fc4), rgb(0x2a62e0)}
	Busy = Style{rgb(0xf59e0b), rgb(0xd97706), rgb(0xc26a05)}
	Off  = Style{rgb(0x9aa3b5), rgb(0x7a8496), rgb(0x8a94a6)}
)

func rgb(v uint32) color.RGBA { return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff} }

// the SVG's geometry, in its 256-unit viewBox

type pt struct{ x, y float64 }

// arc: from a to b, radius r, clockwise on screen (the SVG's sweep=1,
// small arc)
type arc struct {
	c      pt
	r      float64
	a0, da float64 // start angle, and the (positive) angle swept
}

func newArc(a, b pt, r float64) arc {
	m := pt{(a.x + b.x) / 2, (a.y + b.y) / 2}
	dx, dy := b.x-a.x, b.y-a.y
	l := math.Hypot(dx, dy)
	h := math.Sqrt(r*r - l*l/4)
	// sweep=1 small arc: the centre is to the right of a→b (y down)
	c := pt{m.x - dy/l*h, m.y + dx/l*h}
	a0 := math.Atan2(a.y-c.y, a.x-c.x)
	a1 := math.Atan2(b.y-c.y, b.x-c.x)
	da := math.Mod(a1-a0+4*math.Pi, 2*math.Pi)
	return arc{c, r, a0, da}
}

func (s arc) dist(p pt) float64 {
	t := math.Mod(math.Atan2(p.y-s.c.y, p.x-s.c.x)-s.a0+4*math.Pi, 2*math.Pi)
	if t <= s.da {
		return math.Abs(math.Hypot(p.x-s.c.x, p.y-s.c.y) - s.r)
	}
	e0 := pt{s.c.x + s.r*math.Cos(s.a0), s.c.y + s.r*math.Sin(s.a0)}
	e1 := pt{s.c.x + s.r*math.Cos(s.a0+s.da), s.c.y + s.r*math.Sin(s.a0+s.da)}
	return math.Min(math.Hypot(p.x-e0.x, p.y-e0.y), math.Hypot(p.x-e1.x, p.y-e1.y))
}

func segDist(p, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}

var (
	arcs  = []arc{newArc(pt{100, 146}, pt{148, 126}, 28), newArc(pt{156, 150}, pt{108, 170}, 28)}
	heads = [][3]pt{{{150, 112}, {150, 128}, {134, 128}}, {{106, 184}, {106, 168}, {122, 168}}}
)

func within(p, c pt, r float64) bool { return math.Hypot(p.x-c.x, p.y-c.y) <= r }

// inTile: the rounded square from o to o+w, rx 56 at full size
func inTile(p pt, o, w float64) bool {
	if p.x < o || p.y < o || p.x > o+w || p.y > o+w {
		return false
	}
	rx := 56 * w / 256
	cx := math.Max(o+rx, math.Min(o+w-rx, p.x))
	cy := math.Max(o+rx, math.Min(o+w-rx, p.y))
	return within(p, pt{cx, cy}, rx)
}

// inFolder: M52 86 a14 ... h42 l16 16 h66 a14 v84 a14 H66 a14 z
func inFolder(p pt) bool {
	switch {
	case p.x < 52 || p.x > 204 || p.y < 72 || p.y > 200:
		return false
	case p.x < 66 && p.y < 86:
		return within(p, pt{66, 86}, 14)
	case p.y < 88: // the tab, its right side the 45° line from (108, 72)
		return p.x <= 108+(p.y-72)
	case p.x > 190 && p.y < 102:
		return within(p, pt{190, 102}, 14)
	case p.x > 190 && p.y > 186:
		return within(p, pt{190, 186}, 14)
	case p.x < 66 && p.y > 186:
		return within(p, pt{66, 186}, 14)
	}
	return true
}

func onArrows(p pt, half float64) bool {
	for _, a := range arcs {
		if a.dist(p) <= half {
			return true
		}
	}
	for _, h := range heads {
		if segDist(p, h[0], h[1]) <= half || segDist(p, h[1], h[2]) <= half {
			return true
		}
	}
	return false
}

// Draw is the icon at n×n pixels, 4×4 supersampled.
func Draw(s Style, n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	// small (tray) sizes show less margin around the folder
	o, w := 0.0, 256.0
	if n <= 24 {
		o, w = 26, 204
	}
	u := w / float64(n) // viewBox units per pixel
	// the SVG's stroke is 11 units; small icons get at least ~1.6px
	half := math.Max(11, 1.6*u) / 2
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	const ss = 4
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					p := pt{o + (float64(x)+(float64(sx)+.5)/ss)*u, o + (float64(y)+(float64(sy)+.5)/ss)*u}
					if !inTile(p, o, w) {
						continue
					}
					var c color.RGBA
					switch {
					case inFolder(p) && onArrows(p, half):
						c = s.Arrow
					case inFolder(p):
						c = white
					default:
						t := (p.x + p.y - 2*o) / (2 * w)
						c = mix(s.Bg0, s.Bg1, t)
					}
					r, g, b, a = r+float64(c.R), g+float64(c.G), b+float64(c.B), a+1
				}
			}
			if a == 0 {
				continue
			}
			// premultiplied: colour sums over all samples, alpha their share
			k := float64(ss * ss)
			img.SetRGBA(x, y, color.RGBA{uint8(r / k), uint8(g / k), uint8(b / k), uint8(255 * a / k)})
		}
	}
	return img
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + .5) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 0xff}
}

// PNG is the icon as a PNG.
func PNG(s Style, n int) []byte {
	var b bytes.Buffer
	png.Encode(&b, Draw(s, n))
	return b.Bytes()
}

// ICO wraps PNGs (each n×n, n ≤ 256) in an ICO container.
func ICO(sizes []int, s Style) []byte {
	pngs := make([][]byte, len(sizes))
	for i, n := range sizes {
		pngs[i] = PNG(s, n)
	}
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))}) // reserved, icon, count
	off := uint32(6 + 16*len(sizes))
	for i, n := range sizes {
		d := byte(n)
		if n >= 256 {
			d = 0 // 0 means 256
		}
		b.Write([]byte{d, d, 0, 0})                             // w, h, colours, reserved
		binary.Write(&b, binary.LittleEndian, [2]uint16{1, 32}) // planes, bpp
		binary.Write(&b, binary.LittleEndian, [2]uint32{uint32(len(pngs[i])), off})
		off += uint32(len(pngs[i]))
	}
	for _, p := range pngs {
		b.Write(p)
	}
	return b.Bytes()
}
