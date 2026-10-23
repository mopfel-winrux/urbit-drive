// genicon writes Drive's app icon for packaging.
//
//	go run ./scripts/genicon -size 256 drive.png
//	go run ./scripts/genicon -ico drive.ico      (16, 24, 32, 48, 64, 256)
//	go run ./scripts/genicon -sheet sheet.png    every state at tray and app sizes
package main

import (
	"bytes"
	"flag"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"

	"github.com/mopfel-winrux/urbit-drive/client/internal/icon"
)

func main() {
	size := flag.Int("size", 256, "PNG size in pixels")
	ico := flag.Bool("ico", false, "write a multi-size ICO instead")
	sheet := flag.Bool("sheet", false, "write a preview of every state and size")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("usage: genicon [-size N | -ico] out")
	}
	b := icon.PNG(icon.Idle, *size)
	if *ico {
		b = icon.ICO([]int{16, 24, 32, 48, 64, 256}, icon.Idle)
	}
	if *sheet {
		b = previewSheet()
	}
	if err := os.WriteFile(flag.Arg(0), b, 0o644); err != nil {
		log.Fatal(err)
	}
}

// previewSheet: idle, busy and off in rows, at 16, 22, 32, 64 and 128px.
func previewSheet() []byte {
	sizes := []int{16, 22, 32, 64, 128}
	styles := []icon.Style{icon.Idle, icon.Busy, icon.Off}
	w := 8
	for _, n := range sizes {
		w += n + 8
	}
	sheet := image.NewRGBA(image.Rect(0, 0, w, len(styles)*(128+8)+8))
	draw.Draw(sheet, sheet.Bounds(), image.White, image.Point{}, draw.Src)
	for r, st := range styles {
		x := 8
		for _, n := range sizes {
			y := 8 + r*(128+8)
			draw.Draw(sheet, image.Rect(x, y, x+n, y+n), icon.Draw(st, n), image.Point{}, draw.Over)
			x += n + 8
		}
	}
	var b bytes.Buffer
	png.Encode(&b, sheet)
	return b.Bytes()
}
