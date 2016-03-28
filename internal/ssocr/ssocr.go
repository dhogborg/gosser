package ssocr

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

	"image/color"

	// Package image/jpeg is not used explicitly in the code below,
	// but is imported for its initialization side-effect, which allows
	// image.Decode to understand JPEG and PNG formatted images.
	_ "image/jpeg"
	"image/png"
)

// Debug enables debug output
var DEBUG = false

// SSOCR is a seven segment display OCR reader
type SSOCR struct {
	Positions int
	Manifest  []byte
}

// NewSSOCR returns a new scanner with the number of positions
// specified.
func NewSSOCR(positions int, manifest []byte) *SSOCR {
	s := &SSOCR{
		Positions: positions,
		Manifest:  manifest,
	}

	if DEBUG {
		os.Mkdir("./debug", 755)
		log.SetLevel(log.DebugLevel)
	}

	return s
}

// Scan returns the string parsed from the image file.
// The resulting string always contains the same number
// of charaters as the Positions int. In case of read error
// the charater place is marked with a blankspace.
func (s *SSOCR) Scan(imagefile string) string {
	// Decode the JPEG data. If reading from file, create a reader with
	img := s.readFile(imagefile)

	var predefs []*SsDigit
	if s.Manifest != nil {
		err := json.Unmarshal(s.Manifest, &predefs)
		if err != nil {
			panic(err)
		}
	}

	str := ""
	digits := []*SsDigit{}
	for a := 0; a < s.Positions; a++ {

		segm := s.extractPosition(img, a)
		var position *SsDigit

		if len(predefs) == 0 {
			position = NewSsDigit(segm)
		} else {
			position = predefs[a]
			position.Image = segm
		}

		position.Position = a
		position.Scan()
		str = str + position.String()

		digits = append(digits, position)

	}

	if DEBUG {
		lines := []string{}
		for _, digit := range digits {
			lines = digit.AppendTo(lines)
		}
		println(strings.Join(lines, "\n"))
	}

	return str
}

func (s *SSOCR) readFile(path string) image.Image {
	reader, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}

	return img
}

func (s *SSOCR) extractPosition(img image.Image, position int) image.Image {

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	// copy the generic image data to a greyscale image model
	if g, ok := copyImage(img, rgba).(*image.RGBA); ok {
		rgba = g
	}

	segmentWidth := int(float64(bounds.Dx()) / float64(s.Positions))
	segmentInset := int(float64(segmentWidth) * float64(position))

	// extract a subimage with the positions inset
	subrect := image.Rect(bounds.Min.X+segmentInset, bounds.Min.Y, bounds.Min.X+segmentInset+segmentWidth, bounds.Max.Y)
	return rgba.SubImage(subrect)
}

type SsDigit struct {
	Position int
	Image    image.Image

	NorthPoint []int `json:"north"`
	SouthPoint []int `json:"south"`

	// segments
	NorthPole bool
	NorthWest bool
	NorthEast bool

	Equator bool

	SouthWest bool
	SouthEast bool
	SouthPole bool
}

const (
	NORTH = iota
	SOUTH = iota
	WEST  = iota
	EAST  = iota
)

func NewSsDigit(img image.Image) *SsDigit {
	return &SsDigit{
		Image: img,
	}
}

func (s *SsDigit) Scan() {

	b := s.Image.Bounds()
	quarterHeight := int(float64(b.Dy()) / 4.0)
	halfWidth := int(float64(b.Dx()) / 2.0)
	quarterWidth := int(float64(b.Dx()) * 0.4)

	// scan in 4 directions from two points, north and south.

	// North origin points and the center segment
	var northOrigin image.Point
	if len(s.NorthPoint) == 2 {
		northOrigin = image.Point{X: s.NorthPoint[0], Y: s.NorthPoint[1]}
	} else {
		northOrigin = image.Point{X: b.Min.X + halfWidth - 1, Y: b.Min.Y + quarterHeight}
	}
	nBaseValue, _, _, _ := s.Image.At(northOrigin.X, northOrigin.Y).RGBA()

	log.WithFields(log.Fields{
		"pos":        s.Position,
		"X":          northOrigin.X,
		"Y":          northOrigin.Y,
		"predefined": len(s.NorthPoint) == 2,
		"base_value": nBaseValue,
	}).Debug("North origin")

	s.NorthPole = s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterHeight, NORTH))
	s.NorthWest = s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterWidth, WEST))
	s.NorthEast = s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterWidth, EAST))

	// center, use north origin
	s.Equator = s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterHeight*2, SOUTH))

	// South origin points
	// the south is slanted to the left
	var southOrigin image.Point
	if len(s.SouthPoint) == 2 {
		southOrigin = image.Point{X: s.SouthPoint[0], Y: s.SouthPoint[1]}
	} else {
		southOrigin = image.Point{X: b.Min.X + halfWidth - 2, Y: b.Min.Y + quarterHeight*3}
	}
	sBaseValue, _, _, _ := s.Image.At(southOrigin.X, southOrigin.Y).RGBA()

	log.WithFields(log.Fields{
		"pos":        s.Position,
		"X":          southOrigin.X,
		"Y":          southOrigin.Y,
		"predefined": len(s.SouthPoint) == 2,
		"base_value": sBaseValue,
	}).Debug("South origin")

	s.SouthWest = s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterWidth, WEST))
	s.SouthEast = s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterWidth, EAST))
	s.SouthPole = s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterHeight, SOUTH))

	// debug output the segments
	if DEBUG {
		out, err := os.Create(fmt.Sprintf("debug/%d.png", s.Position))
		if err != nil {
			log.Panic(err)
		}
		png.Encode(out, s.Image)
		out.Close()
	}
}

func (s *SsDigit) isActiveSegment(baseValue, crossValue uint32) bool {
	// the base value should be lighter than the cross value.
	// We need about 20% darker value to activate the segment.
	threshold := float64(baseValue) * 0.5
	return float64(crossValue) < threshold
}

func (s *SsDigit) String() string {

	characters := map[string]*SsDigit{
		"0": &SsDigit{NorthPole: true, NorthWest: true, NorthEast: true, SouthWest: true, SouthEast: true, SouthPole: true},
		"1": &SsDigit{NorthEast: true, SouthWest: true},
		"2": &SsDigit{NorthPole: true, NorthEast: true, Equator: true, SouthWest: true, SouthPole: true},
		"3": &SsDigit{NorthPole: true, NorthEast: true, Equator: true, SouthEast: true, SouthPole: true},
		"4": &SsDigit{NorthWest: true, NorthEast: true, Equator: true, SouthEast: true},
		"5": &SsDigit{NorthPole: true, NorthWest: true, Equator: true, SouthEast: true, SouthPole: true},
		"6": &SsDigit{NorthPole: true, NorthWest: true, Equator: true, SouthWest: true, SouthEast: true, SouthPole: true},
		"7": &SsDigit{NorthPole: true, NorthEast: true, SouthEast: true},
		"8": &SsDigit{NorthPole: true, NorthWest: true, NorthEast: true, Equator: true, SouthWest: true, SouthEast: true, SouthPole: true},
		"9": &SsDigit{NorthPole: true, NorthWest: true, NorthEast: true, Equator: true, SouthEast: true},
	}

	for c, d := range characters {
		if s.Equals(d) {
			return c
		}
	}
	return "-"
}

func (s *SsDigit) AppendTo(lines []string) []string {

	if len(lines) != 5 {
		lines = make([]string, 5)
	}

	pl := s.Dotstrings()
	for i := range lines {
		lines[i] += pl[i] + " "
	}
	return lines
}

func (s *SsDigit) Print() {
	b := s.Dotstrings()
	for _, line := range b {
		println(line)
	}
}

func (s *SsDigit) Dotstrings() []string {
	dots := [][]string{
		[]string{" ", " ", " "},
		[]string{" ", " ", " "},
		[]string{" ", " ", " "},
		[]string{" ", " ", " "},
		[]string{" ", " ", " "},
	}

	if s.NorthPole {
		dots[0] = []string{"*", "*", "*"}
	}
	if s.NorthWest {
		dots[0][0] = "*"
		dots[1][0] = "*"
		dots[2][0] = "*"
	}
	if s.NorthEast {
		dots[0][2] = "*"
		dots[1][2] = "*"
		dots[2][2] = "*"
	}
	if s.Equator {
		dots[2] = []string{"*", "*", "*"}
	}
	if s.SouthWest {
		dots[2][0] = "*"
		dots[3][0] = "*"
		dots[4][0] = "*"
	}
	if s.SouthEast {
		dots[2][2] = "*"
		dots[3][2] = "*"
		dots[4][2] = "*"
	}
	if s.SouthPole {
		dots[4] = []string{"*", "*", "*"}
	}

	lines := []string{}
	for _, l := range dots {
		lines = append(lines, strings.Join(l, ""))
	}

	return lines
}

// Equals returns true if the receiver and compration
// object have the same segment configuration
func (s *SsDigit) Equals(comp *SsDigit) bool {
	return (s.NorthPole == comp.NorthPole) &&
		(s.NorthWest == comp.NorthWest) &&
		(s.NorthEast == comp.NorthEast) &&
		(s.Equator == comp.Equator) &&
		(s.SouthWest == comp.SouthWest) &&
		(s.SouthEast == comp.SouthEast) &&
		(s.SouthPole == comp.SouthPole)
}

func (s *SsDigit) minValue(origin image.Point, length, direction int) uint32 {

	// Direction legend:
	// North: origin.Y--
	// West:  origin.X--
	// East:  origin.X++
	// South: origin.Y++
	directionMap := map[int]image.Point{
		NORTH: image.Point{X: 0, Y: -1},
		WEST:  image.Point{X: -1, Y: 0},
		EAST:  image.Point{X: 1, Y: 0},
		SOUTH: image.Point{X: 0, Y: 1},
	}

	bounds := s.Image.Bounds()
	point := origin
	pointMod := directionMap[direction]

	minValue, _, _, _ := s.Image.At(point.X, point.Y).RGBA()
	for a := 0; a < length; a++ {
		point = point.Add(pointMod)

		if point.X < bounds.Min.X || point.X > bounds.Max.X {
			break
		}

		if point.Y < bounds.Min.Y || point.Y > bounds.Max.Y {
			break
		}

		pValue, _, _, _ := s.Image.At(point.X, point.Y).RGBA()
		// minValue = (minValue + pValue) / 2
		if minValue > pValue {
			minValue = pValue
		}

		if DEBUG {
			if img, ok := s.Image.(*image.RGBA); ok {
				img.Set(point.X, point.Y, color.RGBA{uint8(pValue), 255, 0, 255})
			}
		}

	}

	return minValue
}

// imageTarget is fulfilled by most image formats in
// https://golang.org/pkg/image/
type imageTarget interface {
	ColorModel() color.Model
	Set(x int, y int, color color.Color)
}

// copyImage copies the contents of source to target
// converting the color to the target color model.
func copyImage(source image.Image, target imageTarget) imageTarget {
	b := source.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oldColor := source.At(x, y)
			color := target.ColorModel().Convert(oldColor)
			target.Set(x, y, color)
		}
	}

	return target
}
