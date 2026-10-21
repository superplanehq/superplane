#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  blocks = prepare.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  expected_builds = ["Build amd64 Image", "Build arm64 Image", "Build Release Assets"]
  expected_builds.each do |name|
    raise "#{name} must depend only on validation" unless blocks.fetch(name).fetch("dependencies") == ["Validate Release"]
  end
  raise "release assembly must join every build" unless blocks.fetch("Assemble Release").fetch("dependencies").sort == expected_builds.sort
  arm_machine = blocks.fetch("Build arm64 Image").dig("task", "agent", "machine", "type")
  raise "arm64 must use the native arm machine" unless arm_machine == "r1-standard-4"
  raise "SuperPlane release must not build CLI assets" if blocks.key?("Build CLI Assets")
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
  expected_assets = "superplane-sbom.json,superplane-single-host.tar.gz,superplane-${RELEASE_VERSION#v}.tgz,checksums.txt,superplane-amd64.oci.tar,superplane-arm64.oci.tar"
  raise "SuperPlane manifest must list only application assets" unless assemble.include?(expected_assets)
  %w[superplane-cli- install.sh].each do |asset|
    raise "SuperPlane assembly must not include #{asset}" if assemble.include?(asset)
  end
  smoke = blocks.fetch("Smoke Test Single Host")
  raise "smoke test must follow assembly" unless smoke.fetch("dependencies") == ["Assemble Release"]
  smoke_commands = smoke.dig("task", "jobs", 0, "commands").join("\n")
  raise "smoke test must run smoke-test.sh" unless smoke_commands.include?("release/superplane-single-host-tarball/smoke-test.sh")

  publish = YAML.safe_load(File.read(ARGV[1]))
  publish_blocks = publish.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  expected_publishers = ["Publish amd64 Image", "Publish arm64 Image"]
  expected_publishers.each do |name|
    raise "#{name} must depend on verification" unless publish_blocks.fetch(name).fetch("dependencies") == ["Verify Prepared Release"]
  end
  raise "final publication must join both images" unless publish_blocks.fetch("Publish Release").fetch("dependencies").sort == expected_publishers.sort
  publish_commands = publish.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  %w[superplane-cli- install.sh].each do |asset|
    raise "SuperPlane publication must not include #{asset}" if publish_commands.include?(asset)
  end
' "${REPO_ROOT}/.semaphore/prepare-superplane-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-superplane-release.yml"

echo "SuperPlane release pipeline tests passed."
