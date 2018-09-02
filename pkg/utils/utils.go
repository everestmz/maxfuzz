package utils

var supportedBases = map[string]bool{"ubuntu:xenial": true}

// SupportedBase returns true if the base is supported by Maxfuzz
func SupportedBase(base string) bool {
	_, ok := supportedBases[base]
	return ok
}
