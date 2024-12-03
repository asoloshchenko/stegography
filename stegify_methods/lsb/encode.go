package lsb

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" //register jpeg image format
	"image/png"
	"io"

	"github.com/TregubovMY/stegography/bitmanip"
	"github.com/TregubovMY/stegography/utils"
)

// 20 байт выделены для заголовка размера данных
const dataSizeHeaderReservedBytes = 20

//export Encode
func Encode(c []byte, d []byte) ([]byte, error) {
	carrier := io.NopCloser(bytes.NewReader(c))
	data := io.NopCloser(bytes.NewReader(d))

	RGBAImage, format, err := utils.GetImageAsRGBA(carrier)
	if err != nil {
		return nil, fmt.Errorf("ошибка при парсинге изображения-контейнера: %v", err)
	}

	dataBytes := make(chan byte, 128)
	errChan := make(chan error)

	go utils.ReadData(data, dataBytes, errChan)

	imageWidth := RGBAImage.Bounds().Dx()
	imageHeight := RGBAImage.Bounds().Dy()

	hasMoreBytes := true

	var countProcessedBytes int
	var totalNumberBytes uint32

	for x := 0; x < imageWidth && hasMoreBytes; x++ {
		for y := 0; y < imageHeight && hasMoreBytes; y++ {
			// Пропускаем первые 20 байтов для хранения заголовка
			if countProcessedBytes < dataSizeHeaderReservedBytes {
				countProcessedBytes += 4
			} else {
				c := RGBAImage.RGBAAt(x, y)
				hasMoreBytes, err = utils.SetColorSegment(&c.R, dataBytes, errChan)
				if err != nil {
					return nil, err
				}
				if hasMoreBytes {
					totalNumberBytes++
				}
				hasMoreBytes, err = utils.SetColorSegment(&c.G, dataBytes, errChan)
				if err != nil {
					return nil, err
				}
				if hasMoreBytes {
					totalNumberBytes++
				}
				hasMoreBytes, err = utils.SetColorSegment(&c.B, dataBytes, errChan)
				if err != nil {
					return nil, err
				}
				if hasMoreBytes {
					totalNumberBytes++
				}
				RGBAImage.SetRGBA(x, y, c)
			}
		}
	}

	select {
	case _, ok := <-dataBytes:
		if ok {
			return nil, fmt.Errorf("файл данных слишком велик для этого контейнера")
		}
	default:
	}

	setDataSizeHeader(RGBAImage, bitmanip.QuartersOfBytesOf(totalNumberBytes))

	switch format {
	case "png", "jpeg":
		result := new(bytes.Buffer)
		err := png.Encode(result, RGBAImage)
		if err != nil {
			return nil, err
		}
		return result.Bytes(), nil

	default:
		return nil, fmt.Errorf("неподдерживаемый формат контейнера")
	}
}

func setDataSizeHeader(RGBAImage *image.RGBA, dataCountBytes []byte) {
	width := RGBAImage.Bounds().Dx()
	height := RGBAImage.Bounds().Dy()

	count := 0

	for x := 0; x < width && count < (dataSizeHeaderReservedBytes/4)*3; x++ {
		for y := 0; y < height && count < (dataSizeHeaderReservedBytes/4)*3; y++ {
			c := RGBAImage.RGBAAt(x, y)
			c.R = bitmanip.SetLastTwoBits(c.R, dataCountBytes[count])
			c.G = bitmanip.SetLastTwoBits(c.G, dataCountBytes[count+1])
			c.B = bitmanip.SetLastTwoBits(c.B, dataCountBytes[count+2])
			RGBAImage.SetRGBA(x, y, c)

			count += 3
		}
	}
}
