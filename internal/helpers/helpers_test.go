// +build unit

package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Testcase struct {
	in   string
	out  string
	kind string
}

func TestGetenv(t *testing.T) {
	os.Setenv("TEST_ENV", "set")
	result := Getenv("TEST_ENV", "unset")
	assert.Equal(t, result, "set")

	os.Unsetenv("TEST_ENV")
	result = Getenv("TEST_ENV", "unset")
	assert.Equal(t, result, "unset")
}
