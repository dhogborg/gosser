package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/dhogborg/gosser/internal/ssocr"
)

var DEBUG = false

func main() {

	app := cli.NewApp()
	app.Name = "Gosser"
	app.Usage = "Read seven segment display image, output result to sdtout"
	app.Version = "0.0.1"
	app.Author = "github.com/dhogborg"
	app.Email = "d@hogborg.se"

	app.Action = func(c *cli.Context) {

		DEBUG := c.Bool("debug")
		ssocr.DEBUG = DEBUG

		// use a manifest file for segment reading
		var manifest []byte
		if manifestfile := c.String("manifest"); manifestfile != "" {

			if DEBUG {
				log.WithFields(log.Fields{
					"file": manifestfile,
				}).Info("using manifest file")
			}

			buffer, err := ioutil.ReadFile(manifestfile)
			if err != nil {
				panic(err)
			}
			manifest = buffer
		}

		pos := c.Int("positions")

		ssocr := ssocr.NewSSOCR(pos, manifest)
		result := ssocr.Scan(c.String("input"))

		if c.Bool("pedantic") {
			if strings.Index(result, "-") > -1 {
				log.WithFields(log.Fields{
					"result": result,
				}).Error("result is not well formed (pedantic mode)")
			}
			os.Exit(-1)
		}

		if c.String("output") == "int" {

			i, err := strconv.ParseFloat(result, 64)
			if err != nil {
				log.Panic(err)
			}

			if c.Int("div") > 0 {
				i = i / c.Float64("div")
			}

			fmt.Printf("%f\n", i)
			os.Exit(0)
		}

		// default printout
		println(result)

	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "input,i",
			Usage: "input file",
		},
		cli.StringFlag{
			Name:  "manifest,m",
			Usage: "Manifest file with coordinates for segments",
		},
		cli.IntFlag{
			Name:  "positions,p",
			Usage: "Number of digits in the image",
		},
		cli.StringFlag{
			Name:  "output,o",
			Value: "string",
			Usage: "Output type, int or string",
		},
		cli.BoolFlag{
			Name:  "pedantic",
			Usage: "Pedantic mode will output an error rather than let you see a invalid result",
		},
		cli.IntFlag{
			Name:  "div",
			Usage: "Divide the result by a factor (only int output)",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug output",
		},
	}

	app.Run(os.Args)
}
