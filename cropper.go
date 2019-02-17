// this package was influenced by https://github.com/jwagner/smartcrop.js
package cropper

import (
	"errors"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/intwinelabs/logger"

	"golang.org/x/image/draw"
)

var (
	// ErrInvalidDimensions gets returned when the supplied dimensions are invalid
	ErrInvalidDimensions = errors.New("Expect either a height or width")

	skinColor = [3]float64{0.78, 0.57, 0.44}
)

const (
	detailWeight            = 0.2
	skinBias                = 0.01
	skinBrightnessMin       = 0.2
	skinBrightnessMax       = 1.0
	skinThreshold           = 0.8
	skinWeight              = 1.8
	saturationBrightnessMin = 0.05
	saturationBrightnessMax = 0.9
	saturationThreshold     = 0.4
	saturationBias          = 0.2
	saturationWeight        = 0.3
	scoreDownSample         = 8
	step                    = 8
	scaleStep               = 0.1
	minScale                = 0.9
	maxScale                = 1.0
	edgeRadius              = 0.4
	edgeWeight              = -20.0
	outsideImportance       = -0.5
	ruleOfThirds            = true
	prescale                = true
	prescaleMin             = 400.00
)

// Analyzer interface analyzes a image.Image and returns the best possible crop with the given
// width and height returns an error if invalid
type Analyzer interface {
	FindBestCrop(img image.Image, width, height int) (image.Rectangle, error)
	FindBestCropWithFaces(img image.Image, width, height int, faces []image.Rectangle) (image.Rectangle, error)
}

// Score contains values that classify matches
type Score struct {
	Detail     float64
	Saturation float64
	Skin       float64
}

// Crop contains results
type Crop struct {
	image.Rectangle
	Score Score
}

// Config is used to setup a new analyzer
type Config struct {
	Debug  bool
	Logger *logger.Logger
}

type analyzer struct {
	debug  bool
	logger *logger.Logger
	Resizer
}

// NewAnalyzer returns a new Analyzer using the given Resizer.
func NewAnalyzer(conf Config) Analyzer {
	return &analyzer{debug: conf.Debug, logger: conf.Logger, Resizer: NewDefaultResizer()}
}

func (a analyzer) FindBestCrop(img image.Image, width, height int) (image.Rectangle, error) {
	if width == 0 && height == 0 {
		return image.Rectangle{}, ErrInvalidDimensions
	}

	// resize image for faster processing
	scale := math.Min(float64(img.Bounds().Dx())/float64(width), float64(img.Bounds().Dy())/float64(height))
	var lowimg *image.RGBA
	var prescalefactor = 1.0

	if prescale {
		if f := prescaleMin / math.Min(float64(img.Bounds().Dx()), float64(img.Bounds().Dy())); f < 1.0 {
			prescalefactor = f
		}
		if a.debug {
			a.logger.Infof("prescale factor: %.2f", prescalefactor)
		}
		smallimg := a.Resize(img, uint(float64(img.Bounds().Dx())*prescalefactor), 0)
		lowimg = toRGBA(smallimg)
	} else {
		lowimg = toRGBA(img)
	}

	if a.debug {
		writeImage("png", lowimg, "./smartcrop_prescale.png")
	}

	cropWidth, cropHeight := chop(float64(width)*scale*prescalefactor), chop(float64(height)*scale*prescalefactor)
	realMinScale := math.Min(maxScale, math.Max(1.0/scale, minScale))

	if a.debug {
		a.logger.Infof("original resolution: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
		a.logger.Infof("scale: %f, cropw: %f, croph: %f, minscale: %f\n", scale, cropWidth, cropHeight, realMinScale)
	}

	topCrop, err := a.analyze(lowimg, cropWidth, cropHeight, realMinScale)
	if err != nil {
		return topCrop, err
	}

	if prescale == true {
		topCrop.Min.X = int(chop(float64(topCrop.Min.X) / prescalefactor))
		topCrop.Min.Y = int(chop(float64(topCrop.Min.Y) / prescalefactor))
		topCrop.Max.X = int(chop(float64(topCrop.Max.X) / prescalefactor))
		topCrop.Max.Y = int(chop(float64(topCrop.Max.Y) / prescalefactor))
	}

	return topCrop.Canon(), nil
}

func (a analyzer) FindBestCropWithFaces(img image.Image, width, height int, faces []image.Rectangle) (image.Rectangle, error) {
	if width == 0 && height == 0 {
		return image.Rectangle{}, ErrInvalidDimensions
	}

	// resize image for faster processing
	scale := math.Min(float64(img.Bounds().Dx())/float64(width), float64(img.Bounds().Dy())/float64(height))
	var lowimg *image.RGBA
	var prescalefactor = 1.0

	if prescale {
		if f := prescaleMin / math.Min(float64(img.Bounds().Dx()), float64(img.Bounds().Dy())); f < 1.0 {
			prescalefactor = f
		}
		if a.debug {
			a.logger.Infof("prescale factor: %.2f", prescalefactor)
		}
		smallimg := a.Resize(img, uint(float64(img.Bounds().Dx())*prescalefactor), 0)
		lowimg = toRGBA(smallimg)
	} else {
		lowimg = toRGBA(img)
	}

	if a.debug {
		writeImage("png", lowimg, "./smartcrop_prescale.png")
	}

	cropWidth, cropHeight := chop(float64(width)*scale*prescalefactor), chop(float64(height)*scale*prescalefactor)
	realMinScale := math.Min(maxScale, math.Max(1.0/scale, minScale))

	if a.debug {
		a.logger.Infof("original resolution: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
		a.logger.Infof("scale: %f, cropw: %f, croph: %f, minscale: %f\n", scale, cropWidth, cropHeight, realMinScale)
	}

	topCrop, err := a.analyzeWithFaces(lowimg, cropWidth, cropHeight, realMinScale, img.Bounds(), faces)
	if err != nil {
		return topCrop, err
	}

	if prescale == true {
		topCrop.Min.X = int(chop(float64(topCrop.Min.X) / prescalefactor))
		topCrop.Min.Y = int(chop(float64(topCrop.Min.Y) / prescalefactor))
		topCrop.Max.X = int(chop(float64(topCrop.Max.X) / prescalefactor))
		topCrop.Max.Y = int(chop(float64(topCrop.Max.Y) / prescalefactor))
	}

	return topCrop.Canon(), nil
}

func (c Crop) totalScore() float64 {
	return (c.Score.Detail*detailWeight + c.Score.Skin*skinWeight + c.Score.Saturation*saturationWeight) / float64(c.Dx()) / float64(c.Dy())
}

func chop(x float64) float64 {
	if x < 0 {
		return math.Ceil(x)
	}
	return math.Floor(x)
}

func thirds(x float64) float64 {
	x = (math.Mod(x-(1.0/3.0)+1.0, 2.0)*0.5 - 0.5) * 16.0
	return math.Max(1.0-x*x, 0.0)
}

func bounds(l float64) float64 {
	return math.Min(math.Max(l, 0.0), 255)
}

func importance(crop Crop, x, y int) float64 {
	if crop.Min.X > x || x >= crop.Max.X || crop.Min.Y > y || y >= crop.Max.Y {
		return outsideImportance
	}

	xf := float64(x-crop.Min.X) / float64(crop.Dx())
	yf := float64(y-crop.Min.Y) / float64(crop.Dy())

	px := math.Abs(0.5-xf) * 2.0
	py := math.Abs(0.5-yf) * 2.0

	dx := math.Max(px-1.0+edgeRadius, 0.0)
	dy := math.Max(py-1.0+edgeRadius, 0.0)
	d := (dx*dx + dy*dy) * edgeWeight

	s := 1.41 - math.Sqrt(px*px+py*py)
	if ruleOfThirds {
		s += (math.Max(0.0, s+d+0.5) * 1.2) * (thirds(px) + thirds(py))
	}

	return s + d
}

func score(output *image.RGBA, crop Crop) Score {
	width := output.Bounds().Dx()
	height := output.Bounds().Dy()
	score := Score{}

	for y := 0; y <= height-scoreDownSample; y += scoreDownSample {
		for x := 0; x <= width-scoreDownSample; x += scoreDownSample {

			c := output.RGBAAt(x, y)
			r8 := float64(c.R)
			g8 := float64(c.G)
			b8 := float64(c.B)

			imp := importance(crop, int(x), int(y))
			det := g8 / 255.0

			score.Skin += r8 / 255.0 * (det + skinBias) * imp
			score.Detail += det * imp
			score.Saturation += b8 / 255.0 * (det + saturationBias) * imp
		}
	}

	return score
}

func (a analyzer) analyze(img *image.RGBA, cropWidth, cropHeight, realMinScale float64) (image.Rectangle, error) {
	o := image.NewRGBA(img.Bounds())

	now := time.Now()
	edgeDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed edge:", time.Since(now))
	}
	debugOutput(a.debug, o, "edge")

	now = time.Now()
	skinDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed skin:", time.Since(now))
	}
	debugOutput(a.debug, o, "skin")

	now = time.Now()
	saturationDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed sat:", time.Since(now))
	}
	debugOutput(a.debug, o, "saturation")

	now = time.Now()
	var topCrop Crop
	topScore := -1.0
	cs := crops(o, cropWidth, cropHeight, realMinScale)
	if a.debug {
		a.logger.Infoln("Time elapsed crops:", time.Since(now), len(cs))
	}

	now = time.Now()
	for _, crop := range cs {
		nowIter := time.Now()
		crop.Score = score(o, crop)
		if a.debug {
			a.logger.Infoln("Time elapsed single-score:", time.Since(nowIter))
		}
		if crop.totalScore() > topScore {
			topCrop = crop
			topScore = crop.totalScore()
		}
	}
	if a.debug {
		a.logger.Infoln("Time elapsed score:", time.Since(now))
		drawDebugCrop(topCrop, o)
		debugOutput(true, o, "final")
	}

	return topCrop.Rectangle, nil
}

func getFacesRect(r image.Rectangle, origRect image.Rectangle, faces []image.Rectangle) image.Rectangle {
	var minX, minY, maxX, maxY int
	for idx, face := range faces {
		if idx == 0 {
			minX = face.Min.X
			minY = face.Min.Y
			maxX = face.Max.X
			maxY = face.Max.Y
		} else {
			if face.Min.X < minX {
				minX = face.Min.X
			}
			if face.Min.Y < minY {
				minY = face.Min.Y
			}
			if face.Max.X > maxX {
				maxX = face.Max.X
			}
			if face.Max.X > maxX {
				maxY = face.Max.Y
			}
		}
	}

	xMinT := r.Max.X * minX / origRect.Max.X
	yMinT := r.Max.Y * minY / origRect.Max.Y
	xMaxT := r.Max.X * maxX / origRect.Max.X
	yMaxT := r.Max.Y * maxY / origRect.Max.Y

	min := image.Point{X: xMinT, Y: yMinT}
	max := image.Point{X: xMaxT, Y: yMaxT}
	return image.Rectangle{Min: min, Max: max}
}

func centerPoint(r image.Rectangle) image.Point {
	x := r.Max.X / 2
	y := r.Max.Y / 2
	return image.Point{X: x, Y: y}
}

func distance(p1, p2 image.Point) float64 {
	first := math.Pow(math.Abs(float64(p2.X-p1.X)), 2)
	second := math.Pow(math.Abs(float64(p2.Y-p1.Y)), 2)
	return math.Sqrt(first + second)
}

func (a analyzer) analyzeWithFaces(img *image.RGBA, cropWidth, cropHeight, realMinScale float64, origRect image.Rectangle, faces []image.Rectangle) (image.Rectangle, error) {
	o := image.NewRGBA(img.Bounds())
	now := time.Now()
	edgeDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed edge:", time.Since(now))
	}
	debugOutput(a.debug, o, "edge")

	now = time.Now()
	skinDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed skin:", time.Since(now))
	}
	debugOutput(a.debug, o, "skin")

	now = time.Now()
	saturationDetect(img, o)
	if a.debug {
		a.logger.Infoln("Time elapsed sat:", time.Since(now))
	}
	debugOutput(a.debug, o, "saturation")

	now = time.Now()
	var topCrop Crop
	topScore := -10000.0
	cs := crops(o, cropWidth, cropHeight, realMinScale)
	if a.debug {
		a.logger.Infoln("Time elapsed crops:", time.Since(now), len(cs))
	}

	now = time.Now()
	faceRect := getFacesRect(o.Rect, origRect, faces)
	if a.debug {
		a.logger.Infof("Faces: %+v", faces)
		a.logger.Infof("Faces Rect: %+v", faceRect)
	}
	for _, crop := range cs {
		nowIter := time.Now()
		crop.Score = score(o, crop)
		if a.debug {
			a.logger.Infof("Crop: %+v", crop)
			a.logger.Infoln("Time elapsed single-score:", time.Since(nowIter))
		}
		tScore := crop.totalScore()
		a.logger.Infof("%.6f", tScore)
		if len(faces) > 0 {
			if !faceRect.In(crop.Rectangle) {
				continue
			}
		}
		if tScore > topScore {
			topCrop = crop
			topScore = tScore
		}
	}
	if a.debug {
		a.logger.Infof("Final score: %.6f", topScore)
		a.logger.Infoln("Time elapsed score:", time.Since(now))
		drawDebugCrop(topCrop, o)
		debugOutput(true, o, "final")
	}

	// check if we failed making a good choice from faces and math
	if topCrop.Rectangle.Max.X == 0 && topCrop.Rectangle.Max.Y == 0 {
		return a.analyze(img, cropWidth, cropHeight, realMinScale)
	}

	return topCrop.Rectangle, nil
}

func saturation(c color.RGBA) float64 {
	cMax, cMin := uint8(0), uint8(255)
	if c.R > cMax {
		cMax = c.R
	}
	if c.R < cMin {
		cMin = c.R
	}
	if c.G > cMax {
		cMax = c.G
	}
	if c.G < cMin {
		cMin = c.G
	}
	if c.B > cMax {
		cMax = c.B
	}
	if c.B < cMin {
		cMin = c.B
	}

	if cMax == cMin {
		return 0
	}
	maximum := float64(cMax) / 255.0
	minimum := float64(cMin) / 255.0

	l := (maximum + minimum) / 2.0
	d := maximum - minimum

	if l > 0.5 {
		return d / (2.0 - maximum - minimum)
	}

	return d / (maximum + minimum)
}

func cie(c color.RGBA) float64 {
	return 0.5126*float64(c.B) + 0.7152*float64(c.G) + 0.0722*float64(c.R)
}

func skinCol(c color.RGBA) float64 {
	r8, g8, b8 := float64(c.R), float64(c.G), float64(c.B)

	mag := math.Sqrt(r8*r8 + g8*g8 + b8*b8)
	rd := r8/mag - skinColor[0]
	gd := g8/mag - skinColor[1]
	bd := b8/mag - skinColor[2]

	d := math.Sqrt(rd*rd + gd*gd + bd*bd)
	return 1.0 - d
}

func makeCies(img *image.RGBA) []float64 {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	cies := make([]float64, width*height, width*height)
	i := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cies[i] = cie(img.RGBAAt(x, y))
			i++
		}
	}

	return cies
}

func edgeDetect(i *image.RGBA, o *image.RGBA) {
	width := i.Bounds().Dx()
	height := i.Bounds().Dy()
	cies := makeCies(i)

	var lightness float64
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x == 0 || x >= width-1 || y == 0 || y >= height-1 {
				//lightness = cie((*i).At(x, y))
				lightness = 0
			} else {
				lightness = cies[y*width+x]*4.0 -
					cies[x+(y-1)*width] -
					cies[x-1+y*width] -
					cies[x+1+y*width] -
					cies[x+(y+1)*width]
			}

			nc := color.RGBA{0, uint8(bounds(lightness)), 0, 255}
			o.SetRGBA(x, y, nc)
		}
	}
}

func skinDetect(i *image.RGBA, o *image.RGBA) {
	width := i.Bounds().Dx()
	height := i.Bounds().Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lightness := cie(i.RGBAAt(x, y)) / 255.0
			skin := skinCol(i.RGBAAt(x, y))

			c := o.RGBAAt(x, y)
			if skin > skinThreshold && lightness >= skinBrightnessMin && lightness <= skinBrightnessMax {
				r := (skin - skinThreshold) * (255.0 / (1.0 - skinThreshold))
				nc := color.RGBA{uint8(bounds(r)), c.G, c.B, 255}
				o.SetRGBA(x, y, nc)
			} else {
				nc := color.RGBA{0, c.G, c.B, 255}
				o.SetRGBA(x, y, nc)
			}
		}
	}
}

func saturationDetect(i *image.RGBA, o *image.RGBA) {
	width := i.Bounds().Dx()
	height := i.Bounds().Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lightness := cie(i.RGBAAt(x, y)) / 255.0
			saturation := saturation(i.RGBAAt(x, y))

			c := o.RGBAAt(x, y)
			if saturation > saturationThreshold && lightness >= saturationBrightnessMin && lightness <= saturationBrightnessMax {
				b := (saturation - saturationThreshold) * (255.0 / (1.0 - saturationThreshold))
				nc := color.RGBA{c.R, c.G, uint8(bounds(b)), 255}
				o.SetRGBA(x, y, nc)
			} else {
				nc := color.RGBA{c.R, c.G, 0, 255}
				o.SetRGBA(x, y, nc)
			}
		}
	}
}

func crops(i image.Image, cropWidth, cropHeight, realMinScale float64) []Crop {
	res := []Crop{}
	width := i.Bounds().Dx()
	height := i.Bounds().Dy()

	minDimension := math.Min(float64(width), float64(height))
	var cropW, cropH float64

	if cropWidth != 0.0 {
		cropW = cropWidth
	} else {
		cropW = minDimension
	}
	if cropHeight != 0.0 {
		cropH = cropHeight
	} else {
		cropH = minDimension
	}

	for scale := maxScale; scale >= realMinScale; scale -= scaleStep {
		for y := 0; float64(y)+cropH*scale <= float64(height); y += step {
			for x := 0; float64(x)+cropW*scale <= float64(width); x += step {
				res = append(res, Crop{
					Rectangle: image.Rect(x, y, x+int(cropW*scale), y+int(cropH*scale)),
				})
			}
		}
	}

	return res
}

// toRGBA converts an image.Image to an image.RGBA
func toRGBA(img image.Image) *image.RGBA {
	switch img.(type) {
	case *image.RGBA:
		return img.(*image.RGBA)
	}
	out := image.NewRGBA(img.Bounds())
	draw.Copy(out, image.Pt(0, 0), img, img.Bounds(), draw.Src, nil)
	return out
}
