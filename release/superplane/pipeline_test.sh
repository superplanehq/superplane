#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  blocks = prepare.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  expected_builds = ["Build amd64 Image", "Build arm64 Image", "Build Release Assets", "Build CLI Assets"]
  expected_builds.each do |name|
    raise "#{name} must depend only on validation" unless blocks.fetch(name).fetch("dependencies") == ["Validate Release"]
  end
  raise "release assembly must join every build" unless blocks.fetch("Assemble Release").fetch("dependencies").sort == expected_builds.sort
  arm_machine = blocks.fetch("Build arm64 Image").dig("task", "agent", "machine", "type")
  raise "arm64 must use the native arm machine" unless arm_machine == "r1-standard-4"
  cli_parallelism = blocks.fetch("Build CLI Assets").dig("task", "jobs", 0, "parallelism")
  raise "CLI builds must use four independent jobs" unless cli_parallelism == 4
  cli_commands = blocks.fetch("Build CLI Assets").dig("task", "jobs", 0, "commands").join("\n")
  raise "CLI jobs must treat the Semaphore index as 1-based" unless cli_commands.include?("SEMAPHORE_JOB_INDEX - 1")
  %w[amd64 arm64].each do |architecture|
    commands = blocks.fetch("Build #{architecture} Image").dig("task", "jobs", 0, "commands").join("\n")
    archive = "superplane-#{architecture}.oci.tar"
    [archive, "#{archive}.sha256", "#{archive}.descriptor.json"].each do |artifact|
      raise "#{architecture} must publish #{artifact}" unless commands.include?(%(artifact push workflow "#{artifact}"))
    end
  end
  assemble = blocks.fetch("Assemble Release").dig("task", "jobs", 0, "commands").join("\n")
  %w[amd64 arm64].each do |architecture|
    raise "assembly must verify the #{architecture} archive" unless assemble.include?("verify-prepared-archive.sh superplane-#{architecture}.oci.tar")
  end

  publish = YAML.safe_load(File.read(ARGV[1]))
  publish_blocks = publish.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  expected_publishers = ["Publish amd64 Image", "Publish arm64 Image"]
  expected_publishers.each do |name|
    raise "#{name} must depend on verification" unless publish_blocks.fetch(name).fetch("dependencies") == ["Verify Prepared Release"]
  end
  raise "final publication must join both images" unless publish_blocks.fetch("Publish Release").fetch("dependencies").sort == expected_publishers.sort
' "${REPO_ROOT}/.semaphore/prepare-superplane-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-superplane-release.yml"

echo "SuperPlane release pipeline tests passed."
