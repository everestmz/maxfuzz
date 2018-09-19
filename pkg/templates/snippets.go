package templates

//
// BUILD STEPS SNIPPETS
//

var buildStepsPrefix = `
set -x
set -e

cd /root/fuzzer

#### Build environment setup
cp $CORPUS/* /root/fuzz_in/
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
mv /root/third_party /usr/local/bin/third_party
rm -rf /root/TMP_CLANG
rm -rf /root/third_party
cd /root/fuzzer
`

//
// ENVIRONMENT SNIPPETS
//

var environmentPrefix = `
#### Required for build steps:
export BUILD_FILES="/root/fuzzer"

#### Required for run-time
export FUZZER_NAME="%s"
`

var environmentAsanBlock = `
export AFL_USE_ASAN="1"
export ASAN_OPTIONS="symbolize=0:detect_leaks=0:abort_on_error=1"
export ASAN_SYMBOLIZER_PATH="/usr/local/bin/third_party/llvm-build/Release+Asserts/bin/llvm-symbolizer"
`

var pythonEnvironmentSettings = `
export AFL_FUZZ="/usr/local/bin/py-afl-fuzz"
export AFL_BINARY=%s
export AFL_MEMORY_LIMIT=%s
export AFL_OPTIONS="%s"
`

var genericEnvironmentSettings = `
export AFL_FUZZ="/usr/local/bin/afl/afl-fuzz"
export AFL_BINARY=%s
export AFL_MEMORY_LIMIT=%s
export AFL_OPTIONS="%s"
`

var goEnvironmentSettings = `
export GO_FUZZ_ZIP=$BUILD_FILES/%s
`
