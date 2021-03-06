// Package ppm implements a Portable Pixel Map (PPM) image decoder and encoder.
//
// The PPM specification is at http://netpbm.sourceforge.net/doc/ppm.html.
package ppm

import (
	"bufio"
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"strconv"
)

var (
	errBadHeader   = errors.New("ppm: invalid header")
	errNotEnough   = errors.New("ppm: not enough image data")
	errUnsupported = errors.New("ppm: unsupported format (maxVal != 255)")
)

func init() {
	image.RegisterFormat("ppm", "P6", Decode, DecodeConfig)
	image.RegisterFormat("ppm", "P3", Decode, DecodeConfig)
}

// Decode reads a PPM image from Reader r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	var d decoder
	img, err := d.decode(r, false)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// DecodeConfig returns the color model and dimensions of a PPM image without
// decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	var d decoder
	if _, err := d.decode(r, true); err != nil {
		return image.Config{}, err
	}
	return image.Config{
		ColorModel: color.RGBAModel,
		Width:      d.width,
		Height:     d.height,
	}, nil
}

// decoder is the type used to decode a PPM file.
type decoder struct {
	br *bufio.Reader

	// from header
	magicNumber string
	width       int
	height      int
	maxVal      int // 255, TODO: support 0 < maxVal < 65536
}

func (d *decoder) decode(r io.Reader, configOnly bool) (image.Image, error) {
	d.br = bufio.NewReader(r)
	var err error

	// decode header
	err = d.decodeHeader()
	if err != nil {
		return nil, err
	}
	if configOnly {
		return nil, nil
	}

	// decode image
	pixel := make([]byte, 3)

	img := image.NewRGBA(image.Rect(0, 0, d.width, d.height))

	if d.magicNumber == "P6" {
		for y := 0; y < d.height; y++ {
			for x := 0; x < d.width; x++ {
				_, err = io.ReadFull(d.br, pixel)
				if err != nil {
					return nil, errNotEnough
				}
				img.SetRGBA(x, y, color.RGBA{pixel[0], pixel[1], pixel[2], 0xff})
			}
		}
	} else if d.magicNumber == "P3" {
		for y := 0; y < d.height; y++ {
			for x := 0; x < d.width; x++ {
				for s := 0; s < 3; s++ {
					pixel[s], err = d.getSubPixel()
					if err != nil {
						return nil, errNotEnough
					}
				}
				img.SetRGBA(x, y, color.RGBA{pixel[0], pixel[1], pixel[2], 0xff})

			}
		}
	}
	return img, nil
}

func (d *decoder) decodeHeader() error {
	var err error
	var b byte
	header := make([]byte, 0)

	comment := false
	for fields := 0; fields < 4; {
		b, _ = d.br.ReadByte()
		if b == '#' {
			comment = true
		} else if !comment {
			header = append(header, b)
		}
		if comment && b == '\n' {
			comment = false
		} else if !comment && (b == ' ' || b == '\n' || b == '\t') {
			fields++
		}
	}
	headerFields := bytes.Fields(header)

	d.magicNumber = string(headerFields[0])
	if d.magicNumber != "P6" {
		if d.magicNumber != "P3" {
			return errBadHeader
		}
	}
	d.width, err = strconv.Atoi(string(headerFields[1]))
	if err != nil {
		return errBadHeader
	}
	d.height, err = strconv.Atoi(string(headerFields[2]))
	if err != nil {
		return errBadHeader
	}

	d.maxVal, err = strconv.Atoi(string(headerFields[3]))
	if err != nil {
		return errBadHeader
	} else if d.maxVal != 255 {
		return errUnsupported
	}
	return nil
}

func (d *decoder) getSubPixel() (byte, error) {
	var err error
	var b byte
	var val int
	subpix := make([]byte, 0)

	comment := false
	for {
		b, _ = d.br.ReadByte()
		if b == '#' {
			comment = true
		} else if !comment && (b == ' ' || b == '\n' || b == '\t') {
			break
		} else if !comment {
			subpix = append(subpix, b)
		}
		if comment && b == '\n' {
			comment = false
		}
	}
	val, err = strconv.Atoi(string(subpix))
	if err != nil {
		return 0, errNotEnough
	}
	return byte(val), nil
}
