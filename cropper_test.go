package cropper

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intwinelabs/goface"

	"github.com/intwinelabs/logger"
	"github.com/stretchr/testify/assert"
)

var (
	testFile = "tests/prettygirl1.jpg"
)

func smartCrop(img image.Image, width, height int) (image.Rectangle, error) {
	conf := Config{Debug: true, Logger: logger.New()}
	analyzer := NewAnalyzer(conf)
	return analyzer.FindBestCrop(img, width, height)
}

func smartCropWithFaces(img image.Image, width, height int, faces []image.Rectangle) (image.Rectangle, error) {
	conf := Config{Debug: true, Logger: logger.New()}
	analyzer := NewAnalyzer(conf)
	return analyzer.FindBestCropWithFaces(img, width, height, faces)
}

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func TestCrop(t *testing.T) {
	assert := assert.New(t)

	var files []string
	err := filepath.Walk("tests", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.Index(info.Name(), ".jpg") > -1 {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		assert.FailNow(err.Error())
	}

	for _, file := range files {
		fi, _ := os.Open(file)
		defer fi.Close()

		img, _, err := image.Decode(fi)
		if err != nil {
			assert.FailNowf("error: %s, %s", err.Error(), fi.Name())
		}

		topCrop, err := smartCrop(img, 1125, 675)
		if err != nil {
			t.Fatal(err)
		}
		/*expected := image.Rect(464, 24, 719, 279)
		if topCrop != expected {
			t.Fatalf("expected %v, got %v", expected, topCrop)
		}*/

		sub, ok := img.(SubImager)
		if ok {
			cropImage := sub.SubImage(topCrop)
			// cropImage := sub.SubImage(image.Rect(topCrop.X, topCrop.Y, topCrop.Width+topCrop.X, topCrop.Height+topCrop.Y))
			imgName := strings.Split(fi.Name(), ".jpg")[0] + "_cropped.jpg"
			logger.Infoln(imgName)
			writeImage("jpeg", cropImage, imgName)
		} else {
			t.Error(errors.New("No SubImage support"))
		}
	}
}

func TestCropWithFaces(t *testing.T) {
	assert := assert.New(t)

	file := "tests/prettygirl1.jpg"
	fi, _ := os.Open(file)
	defer fi.Close()

	img, _, err := image.Decode(fi)
	if err != nil {
		assert.FailNowf("error: %s, %s", err.Error(), fi.Name())
	}

	rec, err := goface.NewRecognizer("models")
	if err != nil {
		assert.FailNowf("error: %s, %s", err.Error(), fi.Name())
	}

	fs, err := rec.RecognizeFile(file, 10)
	if err != nil {
		assert.FailNowf("error: %s, %s", err.Error(), fi.Name())
	}
	var faces []image.Rectangle
	for _, face := range fs {
		faces = append(faces, face.Rectangle)
	}

	topCrop, err := smartCropWithFaces(img, 1125, 675, faces)
	if err != nil {
		t.Fatal(err)
	}
	/*expected := image.Rect(464, 24, 719, 279)
	if topCrop != expected {
		t.Fatalf("expected %v, got %v", expected, topCrop)
	}*/

	sub, ok := img.(SubImager)
	if ok {
		cropImage := sub.SubImage(topCrop)
		// cropImage := sub.SubImage(image.Rect(topCrop.X, topCrop.Y, topCrop.Width+topCrop.X, topCrop.Height+topCrop.Y))
		imgName := strings.Split(fi.Name(), ".jpg")[0] + "_cropped.jpg"
		logger.Infoln(imgName)
		writeImage("jpeg", cropImage, imgName)
	} else {
		t.Error(errors.New("No SubImage support"))
	}

}

func BenchmarkCrop(b *testing.B) {
	fi, err := os.Open(testFile)
	if err != nil {
		b.Fatal(err)
	}
	defer fi.Close()

	img, _, err := image.Decode(fi)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := smartCrop(img, 250, 250); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEdge(b *testing.B) {
	fi, err := os.Open(testFile)
	if err != nil {
		b.Fatal(err)
	}
	defer fi.Close()

	img, _, err := image.Decode(fi)
	if err != nil {
		b.Fatal(err)
	}

	rgbaImg := toRGBA(img)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := image.NewRGBA(img.Bounds())
		edgeDetect(rgbaImg, o)
	}
}

func BenchmarkImageDir(b *testing.B) {
	files, err := ioutil.ReadDir("./examples")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for _, file := range files {
		if strings.Contains(file.Name(), ".jpg") {
			fi, _ := os.Open("./examples/" + file.Name())
			defer fi.Close()

			img, _, err := image.Decode(fi)
			if err != nil {
				b.Error(err)
				continue
			}

			topCrop, err := smartCrop(img, 220, 220)
			if err != nil {
				b.Error(err)
				continue
			}
			fmt.Printf("Top crop: %+v\n", topCrop)

			sub, ok := img.(SubImager)
			if ok {
				cropImage := sub.SubImage(topCrop)
				// cropImage := sub.SubImage(image.Rect(topCrop.X, topCrop.Y, topCrop.Width+topCrop.X, topCrop.Height+topCrop.Y))
				writeImage("jpeg", cropImage, "/tmp/smartcrop/smartcrop-"+file.Name())
			} else {
				b.Error(errors.New("No SubImage support"))
			}
		}
	}
	// fmt.Println("average time/image:", b.t)
}
