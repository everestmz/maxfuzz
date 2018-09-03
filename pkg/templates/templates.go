package templates

import (
	"bytes"
	"fmt"

	maxfuzz "github.com/everestmz/maxfuzz/pkg/utils"
)

// Constants
var shellPrefix = "#!/bin/bash\n"

type Fuzzer interface {
	BuildSteps() []string
	Environment() []string // List of envar definitions
	Run() string           // The binary or fuzzer zip location
	MemoryLimit() string
	Options() string
}

// New returns a new Template struct
func New(fuzzerName, language string, asan bool, base string) (Template, error) {
	if language == maxfuzz.Go {
		asan = false
	}
	if !maxfuzz.SupportedBase(base) {
		return Template{}, fmt.Errorf("base %s not supported", base)
	}
	return Template{
		FuzzerName: fuzzerName,
		Language:   language,
		ASAN:       asan,
		Base:       base,
	}, nil
}

// Template struct used for generating fuzzer files
type Template struct {
	FuzzerName string
	Language   string
	ASAN       bool
	Base       string
}

// GenerateBuildSteps returns full build steps
func (t Template) GenerateBuildSteps(f Fuzzer) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	switch t.Language {
	case maxfuzz.Go:
		// Do nothing
	default:
		buf.WriteString(buildStepsPrefix)
		if t.ASAN {
			buf.WriteString(asanBuildSteps)
		}
		// Ensure we're running things from build files dir
		buf.WriteString("cd $BUILD_FILES\n")
		for _, line := range f.BuildSteps() {
			buf.WriteString(fmt.Sprintf("%s\n", line))
		}
	}

	return buf
}

// GenerateEnvironment returns full environment
func (t Template) GenerateEnvironment(f Fuzzer) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	buf.WriteString(fmt.Sprintf(
		environmentPrefix, t.FuzzerName, t.FuzzerName),
	)

	if t.ASAN {
		buf.WriteString(environmentAsanBlock)
	}

	switch t.Language {
	case maxfuzz.Go:
		buf.WriteString(fmt.Sprintf(goEnvironmentSettings, f.Run()))
	case maxfuzz.Python:
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
		buf.WriteString(fmt.Sprintf("export %s\n", line))
	}

	return buf
}

// GenerateStartFile returns full start file
func (t Template) GenerateStartFile() bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString(shellPrefix)
	switch t.Language {
	case maxfuzz.Go:
		buf.WriteString(fmt.Sprintf(goStartScript, t.FuzzerName))
	default:
		buf.WriteString(fmt.Sprintf(genericStartScript, t.FuzzerName))
	}

	return buf
}
