# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

if not File.exist?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

git_latest_tag = `git tag -l |sort -V -r |head -n1`
git_latest_tag = git_latest_tag.encode('utf-8').strip
git_timestamp, git_hash = `git log -n1 --first-parent --format=%ct:%H .`.chomp.split(":")
git_timestamp = Time.at(git_timestamp.to_i).utc

Gem::Specification.new do |s|
  s.name        = 'arvados-cli'
  s.version     = "#{git_latest_tag}.#{git_timestamp.strftime('%Y%m%d%H%M%S')}"
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Arvados CLI tools"
  s.description = "Arvados command line tools, git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@curoverse.com'
  #s.bindir      = '.'
  s.licenses    = ['Apache-2.0']
  s.files       = ["bin/arv", "bin/arv-tag", "LICENSE-2.0.txt"]
  s.executables << "arv"
  s.executables << "arv-tag"
  s.required_ruby_version = '>= 2.1.0'
  s.add_runtime_dependency 'arvados', '~> 1.3.0', '>= 1.3.0'
  # Our google-api-client dependency used to be < 0.9, but that could be
  # satisfied by the buggy 0.9.pre*.  https://dev.arvados.org/issues/9213
  s.add_runtime_dependency 'arvados-google-api-client', '~> 0.6', '>= 0.6.3', '<0.8.9'
  s.add_runtime_dependency 'activesupport', '>= 3.2.13', '< 5.1'
  s.add_runtime_dependency 'json', '>= 1.7.7', '<3'
  s.add_runtime_dependency 'optimist', '~> 3.0'
  s.add_runtime_dependency 'andand', '~> 1.3', '>= 1.3.3'
  s.add_runtime_dependency 'oj', '~> 3.0'
  s.add_runtime_dependency 'curb', '~> 0.8'
  s.homepage    =
    'https://arvados.org'
end
