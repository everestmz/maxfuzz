require 'erb'
require 'fileutils'

kind = ARGV[0]
@fuzzer_name = ARGV[1]
target = ARGV[2]

if not ["go", "afl"].include?(kind)
  puts "Please specify fuzzer kind"
  exit 1
end

if File.directory?(@fuzzer_name)
  puts "Fuzzer already exists with this name"
  exit 1
end

fuzzer_location = @fuzzer_name
if target != nil
  fuzzer_location = target + fuzzer_location
end

FileUtils.copy_entry("template/#{kind}", fuzzer_location)

for file in ['start', 'environment', 'README.md', 'docker/docker-compose.yml']
  location = "#{fuzzer_location}/#{file}"
  erb_location = "#{location}.erb"
  f = File.open(erb_location)
  contents = f.read

  renderer = ERB.new(contents)
  result = renderer.result()

  File.open(location, 'w') { |out| out.write(result) }
  FileUtils.rm(erb_location)
end
