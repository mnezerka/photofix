package main

import (
	"fmt"
	_ "golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func diffColors(p1 color.RGBA, p2 color.RGBA) int {
	var diff int = 0
	diff += abs(int(p1.R) - int(p2.R))
	diff += abs(int(p1.G) - int(p2.G))
	diff += abs(int(p1.B) - int(p2.B))
	return diff
}

func avgColors(p1 color.RGBA, p2 color.RGBA) color.RGBA {
	var result color.RGBA
	result.R = uint8((int(p1.R) + int(p2.R)) / 2)
	result.G = uint8((int(p1.G) + int(p2.G)) / 2)
	result.B = uint8((int(p1.B) + int(p2.B)) / 2)
	result.A = 255
	return result
}

func rotatePt(p image.Point, imgSize image.Rectangle, angle uint) image.Point {
	var result image.Point = p

	if angle == 90 {
		result.X = p.Y
		result.Y = imgSize.Max.Y - 1 - p.X
	} else if angle == 270 {
		result.X = imgSize.Max.X - 1 - p.Y
		result.Y = p.X
	}

	return result
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No images to be processed")
		return
	}

	for i := 1; i < len(os.Args); i++ {
		filePath := os.Args[i]

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Println("No such file:", filePath)
			continue
		}

		var fileName = filepath.Base(filePath)
		var fileExt = filepath.Ext(fileName)
		var fileNameRaw = strings.TrimSuffix(fileName, fileExt)
		var fileNameFix = fileNameRaw + "_fix.png"
		var filePathFix = filepath.Join(filepath.Dir(filePath), fileNameFix)

		if !strings.EqualFold(fileExt, ".tif") &&
			!strings.EqualFold(fileExt, ".tiff") &&
			!strings.EqualFold(fileExt, ".png") {
			fmt.Println("Invalid file format", fileExt)
			return
		}

		image := loadImage(filePath)

		fmt.Printf("fixing %s -> %s\n", fileName, fileNameFix)

		if imageFix, err := processImage(image); err == nil {
			saveImage(imageFix, filePathFix)
			fmt.Printf("done.\n")
		} else {
			fmt.Println(err)
		}

	}
}

func loadImage(path string) *image.Image {
	reader, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer reader.Close()
	m, _, err := image.Decode(reader)
	if err != nil {
		fmt.Println(err)
	}
	return &m
}

func saveImage(image *draw.Image, path string) {
	toimg, _ := os.Create(path)
	defer toimg.Close()
	png.Encode(toimg, *image)
}

// 1 - horizontal , 2- vertical_90, 3 - vertical_270
func detectOrientation(image *draw.Image) int {
	bounds := (*image).Bounds()

	// 0 - wrong, 1 - horizontal , 2- vertical
	orientation := 0

	if bounds.Max.X == 3008 || bounds.Max.Y == 2000 {
		orientation = 1
	} else if bounds.Max.Y == 3008 || bounds.Max.X == 2000 {
		// implementation of detection
		orientation = 2
	}

	return orientation
}

func processImage(image *image.Image) (*draw.Image, error) {
	// make image writeable
	newImage := cloneToRGBA(*image)

	var err error

	err = fixLineError(image, &newImage)

	return &newImage, err
}

func cloneToRGBA(src image.Image) draw.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func fixLineError(imgOrig *image.Image, img *draw.Image) error {
	const defectX = 1572
	const defectY = 1451
	const dia = 4
	const shiftX = 10
	const shiftY = 0

	// distance to be used for checking colour of surrounding pixels
	const shiftCheck = 2

	bounds := (*img).Bounds()

	fmt.Printf("Fixing line error\n")

	fmt.Printf("...image dimensions (%d, %d)\n", bounds.Max.X, bounds.Max.Y)

	var portrait bool = true

	if bounds.Max.X == 3008 && bounds.Max.Y == 2000 {
		portrait = true
	} else if bounds.Max.Y == 3008 && bounds.Max.X == 2000 {
		portrait = false
	} else {
		return fmt.Errorf("...invalid image dimensions (%d, %d)", bounds.Max.X, bounds.Max.Y)
	}

	var defect [4]image.Point
	defect[0] = image.Pt(defectX, defectY)
	defect[1] = image.Pt(defectY, bounds.Max.Y-defectX)
	defect[2] = image.Pt(bounds.Max.X-defectX, bounds.Max.Y-defectY)
	defect[3] = image.Pt(bounds.Max.X-defectY, defectX)

	var defectDirection = [4][2]int{
		{0, 1},
		{1, 0},
		{0, -1},
		{-1, 0},
	}

	var diff [4]int
	var green [4]uint

	for angle := 0; angle < 4; angle++ {

		if (angle == 0 || angle == 2) && !portrait {
			continue
		} else if (angle == 1 || angle == 3) && portrait {
			continue
		}

		fmt.Printf("...angle: %d\n", angle*90)
		fmt.Print("......defect pos: ", defect[angle], "\n")
		fmt.Print("......defect direction: ", defectDirection[angle], "\n")

		diff[angle] = 0
		green[angle] = 0
		for i := 0; i < 400; i++ {
			var offsetX = i * defectDirection[angle][0]
			var offsetY = i * defectDirection[angle][1]
			//fmt.Printf("...%d, %d\n", defect[angle].X+offsetX, defect[angle].Y+offsetY)

			var c color.RGBA = (*img).At(defect[angle].X+offsetX, defect[angle].Y+offsetY).(color.RGBA)
			//var c2 color.RGBA = (*img).At(defect[angle].X+offsetX, defect[angle].Y+shiftCheck+offsetY).(color.RGBA)
			//diff[angle] += diffColors(c, c2)
			green[angle] += uint(c.G)
		}
		//fmt.Printf("...diff: %d\n", diff[angle])
		fmt.Printf("......green: %d\n", green[angle])
	}

	// find max green
	var angleFinal int = 0
	var greenMax uint = 0

	for angle := 0; angle < 4; angle++ {
		if green[angle] > greenMax {
			angleFinal = angle
			greenMax = green[angle]
		}
	}

	fmt.Printf("...angle final: %d\n", angleFinal*90)

	// fix line
	var fixPos = defect[angleFinal]

	var sampleDir [2]int
	if defectDirection[angleFinal][0] != 0 {
		sampleDir[1] = 1
	} else {
		sampleDir[0] = 1
	}

	fmt.Print("...fixing line from ", fixPos, "sampling direction: ", sampleDir, "\n")

	for fixPos.In(bounds) {

		/*
		 *  +----+---+---+---+----+
		 *  | 11 | 1 | X | 2 | 22 |
		 *  +----+---+---+---+----+
		 *
		 */

		// sample colors around currently fixed pixel "X"
		color11 := (*imgOrig).At(fixPos.X+sampleDir[0]*2, fixPos.Y+sampleDir[1]*2).(color.RGBA)
		color22 := (*imgOrig).At(fixPos.X+sampleDir[0]*-2, fixPos.Y+sampleDir[1]*-2).(color.RGBA)

		// fix color in center pixel "X"
		(*img).Set(fixPos.X, fixPos.Y, avgColors(color11, color22))

		//fmt.Print("...setting ", fixPos, "to white\n")
		//fmt.Print("...setting ", fixPos.X+sampleDir[0]*1, fixPos.Y+sampleDir[1]*1, "as 1\n")

		// fix color in pixel "1"
		(*img).Set(fixPos.X+sampleDir[0]*1, fixPos.Y+sampleDir[1]*1, color11)

		// fix color in pixel "2"
		(*img).Set(fixPos.X+sampleDir[0]*-1, fixPos.Y+sampleDir[1]*-1, color22)

		fixPos.X += defectDirection[angleFinal][0]
		fixPos.Y += defectDirection[angleFinal][1]
	}

	// fix point

	return nil
}
