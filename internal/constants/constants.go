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
	FuzzerBackupLocation  = "/root/backup.zip"
	FuzzerSyncDirectory   = "/root/sync"
	AFLIOOptions          = "/root/config/afl-io/options"
	// Local file constants
	LocalSyncDirectory   = os.ExpandEnv("$HOME/sync")    // Where the crashes are synced to on root
	LocalTargetDirectory = os.ExpandEnv("$HOME/targets") // Where targets are on the root system
	// Docker Images
	FuzzBoxImageName = "maxfuzz"
)
