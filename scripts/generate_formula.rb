#!/usr/bin/env ruby
require 'erb'
require 'fileutils'

root = File.expand_path('..', __dir__)
template_path = File.join(root, '..', 'homebrew-tap', 'Formula', 'agent-align.rb.erb')
output_path = File.join(root, '..', 'homebrew-tap', 'Formula', 'agent-align.rb')

unless File.exist?(template_path)
  abort "Template not found: #{template_path}"
end

template = File.read(template_path)
ver = ENV['VER'] || abort("VER not set")
darwin_arm_sha = ENV['DARWIN_ARM_SHA'] || ''
darwin_amd_sha = ENV['DARWIN_AMD_SHA'] || ''
linux_amd_sha = ENV['LINUX_AMD_SHA'] || ''
linux_arm_sha = ENV['LINUX_ARM_SHA'] || ''

renderer = ERB.new(template, trim_mode: '-')
result = renderer.result(binding)

# Ensure directory exists
FileUtils.mkdir_p(File.dirname(output_path))
File.write(output_path, result)
puts "Wrote formula to #{output_path} (VER=#{ver})"
