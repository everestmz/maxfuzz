package constants

import (
	"os"
)

var (
	// Maxfuzz
	FuzzerOutputDirectory = "/root/fuzz_out"
	FuzzerLocation        = "/root/fuzzer"
	FuzzerEnvironment     = "/root/fuzzer/environment"
	FuzzerBuildSteps      = "/root/fuzzer/build_steps"
	AFLIOOptions          = "/root/config/afl-io/options"
	// Local file constants
	LocalSyncDirectory   = os.ExpandEnv("$HOME/maxfuzz/sync")    // Where the crashes are synced to on root
	LocalTargetDirectory = os.ExpandEnv("$HOME/maxfuzz/targets") // Where targets are on the root system
	LocalCrashStorage    = os.ExpandEnv("$HOME/maxfuzz/crashes") // Where we save the final crashes & output
	// Docker Images
	FuzzBoxImageName = "maxfuzz"
)
