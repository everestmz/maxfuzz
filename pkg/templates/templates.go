package templates

import (
	"bytes"
	"fmt"
)

// Constants
var shellPrefix = "#!/bin/bash\n"

type Fuzzer interface {
	Environment() []string // List of envar definitions
	Run() string           // The binary or fuzzer zip location
	MemoryLimit() string
	Options() string
}

// New returns a new Template struct
func New(fuzzerName, language string, asan bool) Template {
	if language == "go" {
		asan = false
	}
	return Template{
		FuzzerName: fuzzerName,
		Language:   language,
		ASAN:       asan,
	}
}

// Template struct used for generating fuzzer files
type Template struct {
	FuzzerName string
	Language   string
	ASAN       bool
}

// GenerateBuildSteps returns full build steps
func (t Template) GenerateBuildSteps(customSteps []string) string {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	switch t.Language {
	case "go":
		// Do nothing
	default:
		buf.WriteString(buildStepsPrefix)
		if t.ASAN {
			buf.WriteString(asanBuildSteps)
		}
		for _, line := range customSteps {
			buf.WriteString(fmt.Sprintf("%s\n", line))
		}
	}

	return buf.String()
}

// GenerateEnvironment returns full environment
func (t Template) GenerateEnvironment(f Fuzzer) string {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	buf.WriteString(environmentPrefix)

	if t.ASAN {
		buf.WriteString(environmentAsanBlock)
	}

	switch t.Language {
	case "go":
		buf.WriteString(fmt.Sprintf(goEnvironmentSettings, f.Run()))
	case "python":
		buf.WriteString(
			fmt.Sprintf(
				pythonEnvironmentSettings,
				f.Run(),
				f.MemoryLimit(),
				f.Options(),
			),
		)
	default:
		buf.WriteString(
			fmt.Sprintf(
				genericEnvironmentSettings,
				f.Run(),
				f.MemoryLimit(),
				f.Options(),
			),
		)
	}

	for _, line := range f.Environment() {
		buf.WriteString(fmt.Sprintf("%s\n", line))
	}

	return buf.String()
}

// GenerateStartFile returns full start file
func (t Template) GenerateStartFile() string {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	switch t.Language {
	case "go":
		buf.WriteString(fmt.Sprintf(goStartScript, t.FuzzerName))
	default:
		buf.WriteString(fmt.Sprintf(genericStartScript, t.FuzzerName))
	}
}
