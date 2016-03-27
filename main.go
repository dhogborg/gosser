package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"

	"github.com/dhogborg/gosser/internal/ssocr"
)

func main() {

	app := cli.NewApp()
	app.Name = "Gosser"
	app.Usage = "Read seven segment display image, output result to sdtout"
	app.Version = "0.0.1"
	app.Author = "github.com/dhogborg"
	app.Email = "d@hogborg.se"

	app.Action = func(c *cli.Context) {

		if c.Bool("debug") {
			ssocr.DEBUG = true
		}

		ssocr := ssocr.NewSSOCR(c.Int("positions"))
		result := ssocr.Scan(c.String("input"))

		if c.String("output") == "int" {
			result := strings.Replace(result, "-", "0", -1)

			i, err := strconv.ParseFloat(result, 64)
			if err != nil {
				log.Panic(err)
			}

			if c.Int("div") > 0 {
				i = i / c.Float64("div")
			}

			fmt.Printf("%f\n", i)
			return
		}

		// default printout
		println(result)

	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "input,i",
			Usage: "input file",
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
