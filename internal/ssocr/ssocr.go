package ssocr

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"image/color"

	// Package image/jpeg is not used explicitly in the code below,
	// but is imported for its initialization side-effect, which allows
	// image.Decode to understand JPEG and PNG formatted images.
	_ "image/jpeg"
	"image/png"
)

// DEBUG enables debug output
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

		if len(predefs) > a {
			position = predefs[a]
			position.Image = segm
		} else {
			position = NewSsDigit(segm)
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

	copyImage(img, rgba)

	segmentWidth := int(float64(bounds.Dx()) / float64(s.Positions))
	segmentInset := int(float64(segmentWidth) * float64(position))

	// extract a subimage with the positions inset
	subrect := image.Rect(
		bounds.Min.X+segmentInset,
		bounds.Min.Y,
		bounds.Min.X+segmentInset+segmentWidth,
		bounds.Max.Y,
	)

	return rgba.SubImage(subrect)
}

// Direction identifiers
const (
	North = iota
	South = iota
	West  = iota
	East  = iota
)

// Segment identifiers
const (
	NorthPole = 1 << iota // 1
	NorthWest = 1 << iota // 2
	NorthEast = 1 << iota // 4
	Equator   = 1 << iota // 8
	SouthWest = 1 << iota // 16
	SouthEast = 1 << iota // 32
	SouthPole = 1 << iota // 64
)

// SsDigit has a position (n:th digit in image), a north and a south origin,
// and segments added together for charater matching.
type SsDigit struct {
	Position int
	Image    image.Image

	NorthPoint []int `json:"north"`
	SouthPoint []int `json:"south"`

	Segments int
}

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

	if s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterHeight, North)) {
		s.Segments += NorthPole
	}
	if s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterWidth, West)) {
		s.Segments += NorthWest
	}
	if s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterWidth, East)) {
		s.Segments += NorthEast
	}

	// center, use north origin
	if s.isActiveSegment(nBaseValue, s.minValue(northOrigin, quarterHeight, South)) {
		s.Segments += Equator
	}

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

	if s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterWidth, West)) {
		s.Segments += SouthWest
	}
	if s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterWidth, East)) {
		s.Segments += SouthEast
	}
	if s.isActiveSegment(sBaseValue, s.minValue(southOrigin, quarterHeight, South)) {
		s.Segments += SouthPole
	}

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

	characters := map[int]string{
		NorthPole + NorthEast + NorthWest + SouthEast + SouthWest + SouthPole: "0",
		NorthEast + SouthEast: "1",
		NorthPole + NorthEast + Equator + SouthWest + SouthPole:                         "2",
		NorthPole + NorthEast + Equator + SouthEast + SouthPole:                         "3",
		NorthEast + NorthWest + Equator + SouthEast:                                     "4",
		NorthPole + NorthWest + Equator + SouthEast + SouthPole:                         "5",
		NorthPole + NorthWest + Equator + SouthEast + SouthWest + SouthPole:             "6",
		NorthPole + NorthEast + SouthEast:                                               "7",
		NorthPole + NorthEast + NorthWest + Equator + SouthEast + SouthWest + SouthPole: "8",
		NorthPole + NorthEast + NorthWest + Equator + SouthEast:                         "9",
	}

	if c, ok := characters[s.Segments]; ok {
		return c
	}

	return "-"
}

func (s *SsDigit) AppendTo(lines []string) []string {

	if len(lines) != 5 {
		lines = make([]string, 5)
	}

	pl := s.Dotstrings()
	for i := range lines {
		lines[i] += pl[i] + "  "
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
		{" ", " ", " "},
		{" ", " ", " "},
		{" ", " ", " "},
		{" ", " ", " "},
		{" ", " ", " "},
	}
	if (s.Segments & NorthPole) == NorthPole {
		dots[0] = []string{"*", "*", "*"}
	}
	if (s.Segments & NorthWest) == NorthWest {
		dots[0][0] = "*"
		dots[1][0] = "*"
		dots[2][0] = "*"
	}
	if (s.Segments & NorthEast) == NorthEast {
		dots[0][2] = "*"
		dots[1][2] = "*"
		dots[2][2] = "*"
	}
	if (s.Segments & Equator) == Equator {
		dots[2] = []string{"*", "*", "*"}
	}
	if (s.Segments & SouthWest) == SouthWest {
		dots[2][0] = "*"
		dots[3][0] = "*"
		dots[4][0] = "*"
	}
	if (s.Segments & SouthEast) == SouthEast {
		dots[2][2] = "*"
		dots[3][2] = "*"
		dots[4][2] = "*"
	}
	if (s.Segments & SouthPole) == SouthPole {
		dots[4] = []string{"*", "*", "*"}
	}

	lines := []string{}
	for _, l := range dots {
		lines = append(lines, strings.Join(l, " "))
	}

	return lines
}

func (s *SsDigit) minValue(origin image.Point, length, direction int) uint32 {

	directionMap := map[int]image.Point{
		North: {X: 0, Y: -1},
		West:  {X: -1, Y: 0},
		East:  {X: 1, Y: 0},
		South: {X: 0, Y: 1},
	}

	point := origin
	pointMod := directionMap[direction]
	img := s.Image
	bounds := img.Bounds()

	minValue, _, _, _ := s.Image.At(point.X, point.Y).RGBA()

	for a := 0; a < length; a++ {
		point = point.Add(pointMod)

		if point.X < bounds.Min.X || point.X > bounds.Max.X {
			break
		}

		if point.Y < bounds.Min.Y || point.Y > bounds.Max.Y {
			break
		}

		red, _, _, _ := img.At(point.X, point.Y).RGBA()
		if minValue > red {
			minValue = red
		}

		if DEBUG {
			if img, ok := s.Image.(*image.RGBA); ok {
				img.Set(point.X, point.Y, color.RGBA{uint8(red), 255, 0, 255})
			}
		}
	}

	log.WithFields(log.Fields{
		"value": minValue,
		"dir":   direction,
	}).Debug("minvalue")

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
func copyImage(source image.Image, target imageTarget) {
	b := source.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			oldColor := source.At(x, y)
			color := target.ColorModel().Convert(oldColor)
			target.Set(x, y, color)
		}
	}
}
