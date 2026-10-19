#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  blocks = prepare.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  cli = blocks.fetch("Build CLI Assets")
  raise "CLI builds must depend only on validation" unless cli.fetch("dependencies") == ["Validate Release"]
  raise "CLI builds must use four independent jobs" unless cli.dig("task", "jobs", 0, "parallelism") == 4
  commands = cli.dig("task", "jobs", 0, "commands").join("\n")
  raise "CLI jobs must treat the Semaphore index as 1-based" unless commands.include?("SEMAPHORE_JOB_INDEX - 1")
  %w[darwin-amd64 darwin-arm64 linux-amd64 linux-arm64].each do |platform|
    raise "CLI builds must include #{platform}" unless commands.include?(platform)
  end
  assemble = blocks.fetch("Assemble Release").dig("task", "jobs", 0, "commands").join("\n")
  raise "CLI assembly must store a cli manifest" unless assemble.include?("--component cli")
  raise "CLI manifest must include the install script and binaries" unless assemble.include?("install.sh,superplane-cli-darwin-amd64,superplane-cli-darwin-arm64,superplane-cli-linux-amd64,superplane-cli-linux-arm64,checksums.txt")
  promotion = prepare.fetch("promotions").fetch(0)
  raise "prepare must promote the CLI publish pipeline" unless promotion.fetch("name") == "Publish CLI Release"
  raise "prepare must promote publish-cli-release.yml" unless promotion.fetch("pipeline_file") == "publish-cli-release.yml"
  auto_promote = promotion.dig("auto_promote", "when").to_s
  raise "CLI publish must auto-promote from a passed main result" unless auto_promote.include?("branch = ") && auto_promote.include?("main") && auto_promote.include?("result = ") && auto_promote.include?("passed")

  publish = YAML.safe_load(File.read(ARGV[1]))
  raise "CLI publish must use the component-releases queue" unless publish.dig("queue", "name") == "component-releases"
  publish_commands = publish.fetch("blocks").flat_map { |block|
    block.dig("task", "jobs").flat_map { |job| job.fetch("commands") }
  }.join("\n")
  raise "CLI publish must verify the cli manifest" unless publish_commands.include?("--component cli")
  raise "CLI publish must target the encoded cli tag" unless publish_commands.include?("cli/$RELEASE_VERSION")
  %w[publish-image publish-chart skopeo helm].each do |token|
    raise "CLI publish must not publish an image or a chart" if publish_commands.include?(token)
  end

  workflow = YAML.safe_load(File.read(ARGV[2]))
  promotions = workflow.fetch("promotions").to_h { |item| [item.fetch("name"), item] }
  cli_promotion = promotions.fetch("Prepare CLI Release")
  raise "CI must prepare the CLI release separately" unless cli_promotion.fetch("pipeline_file") == "prepare-cli-release.yml"
  raise "CI must keep the SuperPlane release separate" unless promotions.fetch("Prepare SuperPlane Release").fetch("pipeline_file") == "prepare-superplane-release.yml"
  version = cli_promotion.dig("parameters", "env_vars").find { |var| var.fetch("name") == "RELEASE_VERSION" }
  raise "CLI release promotion must require RELEASE_VERSION" unless version && version.fetch("required") == true
  raise "CLI release version must use vMAJOR.MINOR.PATCH[-PRERELEASE]" unless version.fetch("description").include?("vMAJOR.MINOR.PATCH[-PRERELEASE]")
' "${REPO_ROOT}/.semaphore/prepare-cli-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-cli-release.yml" \
  "${REPO_ROOT}/.semaphore/semaphore.yml"

echo "CLI release pipeline tests passed."
