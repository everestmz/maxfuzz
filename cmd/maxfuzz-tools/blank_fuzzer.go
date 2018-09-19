package main

type BlankFuzzer struct{}

func (f BlankFuzzer) BuildSteps() []string {
	return []string{"", "# Custom build steps here", ""}
}

func (f BlankFuzzer) Environment() []string {
	return []string{}
}

func (f BlankFuzzer) Run() string {
	return "REPLACE_THIS"
}

func (f BlankFuzzer) MemoryLimit() string {
	return "none"
}

func (f BlankFuzzer) Options() string {
	return ""
}

func (f BlankFuzzer) Corpus() string {
	return "corpus"
}
