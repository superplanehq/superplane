#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  commands = prepare.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  raise "Fleet Manager prepare must include the docker image archive" unless commands.include?("fleet-manager-docker.oci.tar")

  publish = YAML.safe_load(File.read(ARGV[1]))
  publish_commands = publish.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  raise "Fleet Manager publish must verify the docker image archive" unless publish_commands.include?("fleet-manager-docker.oci.tar")
  raise "Fleet Manager publish must tag the docker image" unless publish_commands.include?("${RELEASE_VERSION}-docker")
  chart = publish_commands.index("release/fleet-manager/publish-chart.sh")
  stage = publish_commands.index("stage-github-release")
  publish_release = publish_commands.index("publish-github-release")
  raise "chart push must finish before the GitHub release is staged" unless chart && stage && chart < stage
  raise "the GitHub release must be published after it is staged" unless stage && publish_release && stage < publish_release
' "${REPO_ROOT}/.semaphore/prepare-fleet-manager-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-fleet-manager-release.yml"

echo "Fleet Manager release pipeline tests passed."
