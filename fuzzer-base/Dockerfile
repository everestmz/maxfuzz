FROM ubuntu:trusty@sha256:ed49036f63459d6e5ed6c0f238f5e94c3a0c70d24727c793c48fded60f70aa96

MAINTAINER Everest Munro-Zeisberger

WORKDIR /root

########################
# SETUP ENV & VERSIONS #
########################

# Versions:
ENV AFL_VERSION 2.52b
ENV RUBY_VERSION 2.3.3
ENV GO_DEP_VERSION 0.4.1

# Environment Variables:
ENV GOPATH=/root/go/
ENV GOBIN=/root/go/bin

################
# INSTALL DEPS #
################

RUN apt-get update
RUN apt-get install -y software-properties-common
RUN apt-add-repository -y ppa:rael-gc/rvm
RUN apt-add-repository -y ppa:gophers/archive
RUN apt-get update --fix-missing
RUN apt-get install -y git
RUN apt-get install -y wget
RUN apt-get install -y gcc
RUN apt-get install -y autoconf
RUN apt-get install -y make
RUN apt-get install -y bison
RUN apt-get install -y libssl-dev
RUN apt-get install -y libreadline-dev
RUN apt-get install -y zlib1g-dev
RUN apt-get install -y pkg-config
RUN apt-get install -y gcc
RUN apt-get install -y clang
RUN apt-get install -y llvm
RUN apt-get install -y rvm
RUN apt-get install -y watch
RUN apt-get install -y cmake
RUN apt-get install -y gdb
RUN apt-get install -y python-virtualenv
RUN apt-get install -y golang-1.9-go
RUN apt-get install -y cython
RUN apt-get install -y build-essential
RUN apt-get install -y libgtk2.0-dev
RUN apt-get install -y libtbb-dev
RUN apt-get install -y python-dev
RUN apt-get install -y python-numpy
RUN apt-get install -y python-scipy
RUN apt-get install -y libjasper-dev
RUN apt-get install -y libjpeg-dev
RUN apt-get install -y libpng-dev
RUN apt-get install -y libtiff-dev
RUN apt-get install -y libavcodec-dev
RUN apt-get install -y libavutil-dev
RUN apt-get install -y libavformat-dev
RUN apt-get install -y libswscale-dev
RUN apt-get install -y libdc1394-22-dev
RUN apt-get install -y libv4l-dev

###########################
# AFL Compilation & Setup #
###########################

# Download AFL and uncompress
RUN wget http://lcamtuf.coredump.cx/afl/releases/afl-$AFL_VERSION.tgz
RUN tar -xvf afl-$AFL_VERSION.tgz
RUN mv afl-$AFL_VERSION afl

# Inject our own AFL config header file
RUN rm /root/afl/config.h
COPY ./config/afl_config/config.h /root/afl/config.h

# Compile both standard gcc, clang etc as well as afl-clang-fast, used for
# faster & persistent test harnesses. Also build the afl-fuzz binary
RUN cd ~/afl && make
RUN cd ~/afl/llvm_mode && make

# Environment Setup
ENV AFL_I_DONT_CARE_ABOUT_MISSING_CRASHES="1"

# Install py-afl-fuzz (for fuzzing python libraries)
RUN git clone https://github.com/jwilk/python-afl.git
RUN cd python-afl && python setup.py install

# Compile ruby from sources with afl, and setup cflags to access instrumented
# ruby headers (useful for ruby library fuzzing with C harneses)
RUN CC=~/afl/afl-clang-fast /usr/share/rvm/bin/rvm install --disable-binary $RUBY_VERSION
ENV LD_LIBRARY_PATH="LD_LIBRARY_PATH=/usr/share/rvm/rubies/ruby-$RUBY_VERSION/lib"
ENV PATH="/usr/share/rvm/rubies/ruby-$RUBY_VERSION/bin:$PATH"
COPY ./config/ ./config/
RUN PKG_CONFIG_PATH=/usr/share/rvm/rubies/ruby-$RUBY_VERSION/lib/pkgconfig pkg-config --cflags --libs ruby-2.3 > ~/config/afl-ruby-flags

# File structure setup
RUN mkdir ~/fuzz_out
RUN mkdir ~/fuzz_in

############################
#SIDECAR & MONITORING SETUP#
############################

# Setup logging and scripts & install Go libs
RUN mkdir /root/logs
WORKDIR /root/go/src/maxfuzz/fuzzer-base
RUN wget https://github.com/golang/dep/releases/download/v$GO_DEP_VERSION/dep-linux-amd64
RUN mv dep-linux-amd64 /usr/local/bin/dep
ENV PATH=/usr/lib/go-1.9/bin/:$PATH
RUN chmod +x /usr/local/bin/dep
RUN go get -u github.com/dvyukov/go-fuzz/...

# Copy Go files into container & compile binaries
RUN mkdir -p /root/go/src/maxfuzz/fuzzer-base
WORKDIR /root/go/src/maxfuzz/fuzzer-base
COPY ./cmd /root/go/src/maxfuzz/fuzzer-base/cmd
COPY ./internal /root/go/src/maxfuzz/fuzzer-base/internal
COPY ./Gopkg.lock /root/go/src/maxfuzz/fuzzer-base/Gopkg.lock
COPY ./Gopkg.toml /root/go/src/maxfuzz/fuzzer-base/Gopkg.toml
COPY ./Makefile /root/go/src/maxfuzz/fuzzer-base/Makefile
RUN make && make install
WORKDIR /root

#################################
# FINAL SETUP & COPYING SCRIPTS #
#################################

RUN mkdir fuzzer-files
COPY ./scripts ./scripts
COPY ./fuzzer-files/base ./fuzzer-files/base
RUN chmod -R 755 /root/fuzzer-files
RUN chmod 755 /root/scripts/reproduce_stdin
RUN echo "export GIT_SHA=test_environment" >> /root/fuzzer-files/base/environment
