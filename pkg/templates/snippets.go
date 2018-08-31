package templates

//
// BUILD STEPS SNIPPETS
//

var buildStepsPrefix = `
#### Build environment setup
cp $BUILD_FILES/examples/* /root/fuzz_in/
cd /root
`

var asanBuildSteps = `
#### Sanitizer Setup:
#### Gives access too LLVM Symbolizer, needed for fuzzing with sanitizers
mkdir /root/TMP_CLANG
cd /root/TMP_CLANG
git clone https://chromium.googlesource.com/chromium/src/tools/clang
cd ..
TMP_CLANG/clang/scripts/update.py
cd /root
`

//
// ENVIRONMENT SNIPPETS
//

var environmentPrefix = `
source /root/fuzzer-files/base/environment
#### Required for build steps:
export BUILD_FILES="/root/fuzzer-files/%s"

#### Required for run-time
export FUZZER_NAME="%s"
`

var environmentAsanBlock = `
export AFL_USE_ASAN="1"
export ASAN_OPTIONS="symbolize=0:detect_leaks=0:abort_on_error=1"
export ASAN_SYMBOLIZER_PATH="/root/third_party/llvm-build/Release+Asserts/bin/llvm-symbolizer"
`

var pythonEnvironmentSettings = `
export AFL_FUZZ="py-afl-fuzz"
export AFL_BINARY=%s
export AFL_MEMORY_LIMIT=%s
export AFL_OPTIONS="%s"
`

var genericEnvironmentSettings = `
export AFL_FUZZ="/root/afl/afl-fuzz"
export AFL_BINARY=%s
export AFL_MEMORY_LIMIT=%s
export AFL_OPTIONS="%s"
`

var goEnvironmentSettings = `
export GO_FUZZ_ZIP=$BUILD_FILES/%s
`

//
// START SNIPPETS
//

var genericStartScript = `
source /root/fuzzer-files/%s/environment
/root/fuzzer-files/base/start $1
`

var goStartScript = `
source /root/fuzzer-files/%s/environment
/root/fuzzer-files/base/start gofuzz
`
