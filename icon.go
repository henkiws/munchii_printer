package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// getIconBytes returns a valid ICO file (with a 32x32 PNG frame inside)
// that Windows CreateIconFromResourceEx accepts.
func getIconBytes() []byte {
	img := makeIconImage(32)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return fallbackICO()
	}
	pngData := pngBuf.Bytes()

	// ICO file layout:
	//   ICONDIR        6 bytes
	//   ICONDIRENTRY  16 bytes
	//   <PNG data>
	var ico bytes.Buffer
	binary.Write(&ico, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // type: 1=ICO
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // image count

	// ICONDIRENTRY
	ico.WriteByte(32) // width
	ico.WriteByte(32) // height
	ico.WriteByte(0)  // color count (0=truecolor)
	ico.WriteByte(0)  // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1))           // planes
	binary.Write(&ico, binary.LittleEndian, uint16(32))          // bit depth
	binary.Write(&ico, binary.LittleEndian, uint32(len(pngData))) // data size
	binary.Write(&ico, binary.LittleEndian, uint32(22))           // data offset (6+16)

	ico.Write(pngData)
	return ico.Bytes()
}

func makeIconImage(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	bodyBlue := color.NRGBA{0x00, 0x78, 0xD4, 0xFF}
	bodyDark := color.NRGBA{0x00, 0x50, 0x9E, 0xFF}
	bodyLight := color.NRGBA{0x33, 0x99, 0xFF, 0xFF}
	paperW := color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	paperE := color.NRGBA{0xBB, 0xBB, 0xBB, 0xFF}
	inkLine := color.NRGBA{0x22, 0x22, 0x22, 0xFF}
	greenLED := color.NRGBA{0x00, 0xDD, 0x66, 0xFF}
	trayBlue := color.NRGBA{0x44, 0x88, 0xBB, 0xFF}

	fill := func(x0, y0, x1, y1 int, c color.NRGBA) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if x >= 0 && x < size && y >= 0 && y < size {
					img.SetNRGBA(x, y, c)
				}
			}
		}
	}

	// Paper input tray (top)
	fill(9, 3, 22, 8, trayBlue)
	for x := 9; x <= 22; x++ {
		img.SetNRGBA(x, 3, bodyLight)
		img.SetNRGBA(x, 8, bodyDark)
	}

	// Printer body
	fill(2, 9, 29, 21, bodyBlue)
	fill(2, 9, 29, 10, bodyLight)
	fill(2, 20, 29, 21, bodyDark)
	for y := 9; y <= 21; y++ {
		img.SetNRGBA(2, y, bodyLight)
		img.SetNRGBA(29, y, bodyDark)
	}

	// Paper feed slot
	fill(5, 14, 26, 15, bodyDark)

	// LED indicator
	fill(23, 11, 26, 13, greenLED)

	// Paper coming out bottom
	fill(8, 17, 23, 31, paperW)
	for y := 17; y <= 31; y++ {
		img.SetNRGBA(8, y, paperE)
		img.SetNRGBA(23, y, paperE)
	}
	for x := 8; x <= 23; x++ {
		img.SetNRGBA(x, 31, paperE)
	}

	// Print lines on paper
	fill(10, 20, 21, 20, inkLine)
	fill(10, 22, 21, 22, inkLine)
	fill(10, 24, 17, 24, inkLine)
	fill(10, 26, 21, 26, inkLine)
	fill(10, 28, 15, 28, inkLine)

	return img
}

func fallbackICO() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	pngData := pngBuf.Bytes()
	var ico bytes.Buffer
	binary.Write(&ico, binary.LittleEndian, uint16(0))
	binary.Write(&ico, binary.LittleEndian, uint16(1))
	binary.Write(&ico, binary.LittleEndian, uint16(1))
	ico.WriteByte(1)
	ico.WriteByte(1)
	ico.WriteByte(0)
	ico.WriteByte(0)
	binary.Write(&ico, binary.LittleEndian, uint16(1))
	binary.Write(&ico, binary.LittleEndian, uint16(32))
	binary.Write(&ico, binary.LittleEndian, uint32(len(pngData)))
	binary.Write(&ico, binary.LittleEndian, uint32(22))
	ico.Write(pngData)
	return ico.Bytes()
}