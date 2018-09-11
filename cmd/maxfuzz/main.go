package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/everestmz/maxfuzz/pkg/templates"

	"github.com/everestmz/maxfuzz/pkg/utils"
	cli "gopkg.in/urfave/cli.v1"
)

func newFuzzer(c *cli.Context) error {
	fuzzerName := c.Args().Get(0)
	dir := c.Args().Get(1)
	language := c.String("lang")
	base := c.String("base")

	// Verify inputs
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	if !utils.SupportedLanguage(language) {
		return fmt.Errorf("language %s not supported", language)
	}
	if !utils.SupportedBase(base) {
		return fmt.Errorf("base %s not supported", base)
	}

	log.Println(fmt.Sprintf("Creating new fuzzer in %s", dir))
	// Setup templates
	log.Println("Templating...")
	f := BlankFuzzer{}
	template, err := templates.New(fuzzerName, language, c.Bool("asan"), base)
	if err != nil {
		return err
	}

	// Write Start File
	log.Println("Writing start file...")
	startFileBuffer := template.GenerateStartFile()
	err = ioutil.WriteFile(filepath.Join(dir, "start"), startFileBuffer.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("error writing start file: %s", err.Error())
	}

	// Write build_steps
	log.Println("Writing build steps...")
	buildStepsFileBuffer := template.GenerateBuildSteps(f)
	err = ioutil.WriteFile(filepath.Join(dir, "build_steps"), buildStepsFileBuffer.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("error writing build_steps file: %s", err.Error())
	}

	// Write environment file
	log.Println("Writing environment...")
	environmentFileBuffer := template.GenerateEnvironment(f)
	err = ioutil.WriteFile(filepath.Join(dir, "environment"), environmentFileBuffer.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("error writing environment file: %s")
	}

	// Make corpus directory
	log.Println("Making corpus directory...")
	os.MkdirAll(filepath.Join(dir, "corpus"), 0755)

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "maxfuzz"
	app.Usage = "do fuzzery things"

	app.Commands = []cli.Command{
		{
			Name:      "new",
			Aliases:   []string{"n"},
			Usage:     "create a new fuzzer",
			Action:    newFuzzer,
			ArgsUsage: "[fuzzer name] [target directory]",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "lang",
					Value: "c",
					Usage: "programming language",
				},
				cli.StringFlag{
					Name:  "base",
					Value: "ubuntu:xenial",
					Usage: "base maxfuzz image to use",
				},
				cli.BoolFlag{
					Name:  "asan",
					Usage: "set this to fuzz with asan",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
