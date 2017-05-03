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
	const shiftCheck = 3

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

	var diff [4]int

	for angle := 0; angle < 4; angle++ {

		if (angle == 0 || angle == 2) && !portrait {
			continue
		} else if (angle == 1 || angle == 3) && portrait {
			continue
		}

		fmt.Printf("...angle: %d\n", angle*90)
		fmt.Print("...defect pos: ", defect[angle], "\n")

		diff[angle] = 0
		for i := 0; i < 10; i++ {
			var c color.RGBA = (*img).At(defect[angle].X+i, defect[angle].Y).(color.RGBA)
			var c2 color.RGBA = (*img).At(defect[angle].X+i, defect[angle].Y+shiftCheck).(color.RGBA)
			diff[angle] += diffColors(c, c2)
		}
		fmt.Printf("...diff: %d\n", diff[angle])
	}

	/*
			var diffLeft int = 0
			var diffRight int = 0

			// take 10 pixel line both from right and left, compute color diff
			// and do heuristic decision based on this diff value

			var defect90 = image.Pt(defectY, bounds.Max.Y-defectX)
			var defect270 = image.Pt(bounds.Max.X-defectY, defectX)

			fmt.Print("...defect 90: ", defect90, "\n")
			fmt.Print("...defect 270: ", defect270, "\n")

			for i := 0; i < 10; i++ {
				var l color.RGBA = (*img).At(defect90.X+i, defect90.Y).(color.RGBA)
				var l2 color.RGBA = (*img).At(defect90.X+i, defect90.Y+shiftCheck).(color.RGBA)
				var r color.RGBA = (*img).At(defect270.X+i, defect270.Y).(color.RGBA)
				var r2 color.RGBA = (*img).At(defect270.X+i, defect270.Y+shiftCheck).(color.RGBA)

				diffLeft += diffColors(l, l2)
				diffRight += diffColors(r, r2)
			}

			if diffLeft > diffRight {
				angle = 90
				fmt.Print("left rotation detected\n")
			} else {
				angle = 270
				fmt.Print("right rotation detected\n")
			}
		}
	*/
	// fix line
	/*
		for y := defectY + 5; y <= 2000; y++ {
			for x := 2; x >= 0; x-- {
				pt := rotatePt(image.Pt(defectX+x, y), bounds, angle)
				ptPick := rotatePt(image.Pt(pt.X+3, pt.Y), bounds, angle)
				colorPick := (*imgOrig).At(ptPick.X, ptPick.Y)
				(*img).Set(pt.X, pt.Y, colorPick)
			}

			for x := -2; x < 0; x++ {
				pt := rotatePt(image.Pt(defectX+x, y), bounds, angle)
				ptPick := rotatePt(image.Pt(pt.X-2, pt.Y), bounds, angle)
				colorPick := (*imgOrig).At(ptPick.X, ptPick.Y)
				(*img).Set(pt.X, pt.Y, colorPick)
			}
	*/
	/*

		        // pick left and right colors
		        ptLeftPick := rotatePt(image.Pt(lineX-2, y), bounds, angle)
		        ptRightPick := rotatePt(image.Pt(lineX+2, y), bounds, angle)
		        left  := (*img).At(ptLeftPick.X, ptLeftPick.Y).(color.RGBA)
		        right := (*img).At(ptRightPick.X, ptRightPick.Y).(color.RGBA)

				// simple interpolation of 3 pixels
		        ptCenter := rotatePt(image.Pt(lineX, y), bounds, angle)
		        ptLeft := rotatePt(image.Pt(lineX - 1, y), bounds, angle)
		        ptRight := rotatePt(image.Pt(lineX + 1, y), bounds, angle)

				(*img).Set(ptCenter.X, ptCenter.Y, color.RGBA{(left.R + right.R) / 2, (left.G + right.G) / 2, (left.B + right.B) / 2, 255})
				(*img).Set(ptLeft.X, ptLeft.Y, color.RGBA{(left.R + right.R) / 2, (left.G + right.G) / 2, (left.B + right.B) / 2, 255})
				(*img).Set(ptRight.X, ptRight.Y, color.RGBA{(left.R + right.R) / 2, (left.G + right.G) / 2, (left.B + right.B) / 2, 255})
	*/
	/*}*/

	// fix point
	/*
		for x := lineX - dia; x <= lineX+dia; x++ {
			for y := lineY - dia; y <= lineY+dia; y++ {
				pt := rotatePt(image.Pt(x, y), bounds, angle)
				ptShifted := rotatePt(image.Pt(x+shiftX, y+shiftY), bounds, angle)
				src := (*img).At(ptShifted.X, ptShifted.Y).(color.RGBA)
				(*img).Set(pt.X, pt.Y, color.RGBA{src.R, src.G, src.B, 255})
			}
		}
	*/
	return nil
}
