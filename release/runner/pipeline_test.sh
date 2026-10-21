#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  commands = prepare.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  raise "runner prepare must include the container image" unless commands.include?("runner.oci.tar")

  publish = YAML.safe_load(File.read(ARGV[1]))
  publish_commands = publish.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  raise "runner publish must push the runner image" unless publish_commands.include?("release/runner/publish-image.sh")
  asset_lines = publish_commands.lines.grep(/--assets /)
  raise "runner manifest must list runner.oci.tar" unless asset_lines.any? { |line| line.include?("runner.oci.tar") }
  raise "runner GitHub assets must omit runner.oci.tar" unless asset_lines.any? { |line| line.include?("--assets") && !line.include?("runner.oci.tar") }
' "${REPO_ROOT}/.semaphore/prepare-runner-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-runner-release.yml"

echo "Runner release pipeline tests passed."
