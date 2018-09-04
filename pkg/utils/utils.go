package utils

var supportedBases = map[string]bool{"ubuntu:xenial": true}
var supportedLanguages = map[string]bool{
	C:      true,
	CPP:    true,
	Go:     true,
	Ruby:   true,
	Python: true,
}

// SupportedBase returns true if the base is supported by Maxfuzz
func SupportedBase(base string) bool {
	_, ok := supportedBases[base]
	return ok
}

// SupportedLanguage returns true if the language is supported by Maxfuzz
func SupportedLanguage(language string) bool {
	_, ok := supportedLanguages[language]
	return ok
}
