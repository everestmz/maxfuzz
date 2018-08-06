# Setting up your new fuzzer

1) Place all needed files in this directory - they will be copied into the
Docker container

2) Put all compile & build steps in `build_steps` under the correct header.
Reference this directory containing all your files using: `$BUILD_FILES`

3) Edit `environment`, which sets all environment variables for the Docker
container. At the very least, change:
- `AFL_BINARY` to match the name of your compiled test harness
- `AFL_MEMORY_LIMIT` to ensure you're providing enough memory to your fuzzer

4) Uncomment the corresponding lines in `environment` and `build_steps` if you
want to fuzz with ASAN

5) To run the fuzzer, call: `/root/fuzzer-files/use-after-free/start` from
within the docker container
