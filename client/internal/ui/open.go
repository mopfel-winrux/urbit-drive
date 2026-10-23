package ui

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os/exec"
	"runtime"
)

// Open shows a URL or a folder with the desktop's default handler.
func Open(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

// Icons, drawn here so there are no binary assets: a rounded square
// with a white "drive" bar, in a colour per state. Windows wants ICO
// (a PNG inside one does), everyone else PNG.
var (
	iconIdle = trayIcon(color.RGBA{0x2b, 0x6c, 0xd8, 0xff})
	iconBusy = trayIcon(color.RGBA{0xe0, 0x8a, 0x1e, 0xff})
	iconOff  = trayIcon(color.RGBA{0x88, 0x88, 0x88, 0xff})
)

func trayIcon(c color.RGBA) []byte {
	const n = 32
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			// rounded corners, radius 6
			dx, dy := max(6-x, x-(n-7), 0), max(6-y, y-(n-7), 0)
			if dx*dx+dy*dy > 36 {
				continue
			}
			px := c
			if y >= 18 && y <= 22 && x >= 7 && x <= 24 {
				px = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			if y == 20 && x >= 20 && x <= 22 {
				px = c // the drive's light
			}
			img.SetRGBA(x, y, px)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	if runtime.GOOS == "windows" {
		return icoOf(b.Bytes(), n)
	}
	return b.Bytes()
}

// icoOf wraps a PNG in a one-image ICO container.
func icoOf(p []byte, size int) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [3]uint16{0, 1, 1}) // reserved, type icon, count
	b.Write([]byte{byte(size), byte(size), 0, 0})              // w, h, colours, reserved
	binary.Write(&b, binary.LittleEndian, [2]uint16{1, 32})    // planes, bpp
	binary.Write(&b, binary.LittleEndian, [2]uint32{uint32(len(p)), 22})
	b.Write(p)
	return b.Bytes()
}
