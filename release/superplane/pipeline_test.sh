#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ruby -ryaml -e '
  prepare = YAML.safe_load(File.read(ARGV[0]))
  blocks = prepare.fetch("blocks").to_h { |block| [block.fetch("name"), block] }
  expected_builds = ["Build amd64 Image", "Build arm64 Image", "Build Release Assets"]
  raise "candidate verification must depend on validation" unless blocks.fetch("Verify Release Candidate").fetch("dependencies") == ["Validate Release"]
  expected_builds.each do |name|
    raise "#{name} must depend on candidate verification" unless blocks.fetch(name).fetch("dependencies") == ["Verify Release Candidate"]
  end
  raise "release assembly must join every build" unless blocks.fetch("Assemble Release").fetch("dependencies").sort == expected_builds.sort
  arm_machine = blocks.fetch("Build arm64 Image").dig("task", "agent", "machine", "type")
  raise "arm64 must use the native arm machine" unless arm_machine == "r1-standard-4"
  raise "SuperPlane release must not build CLI assets" if blocks.key?("Build CLI Assets")
  %w[amd64 arm64].each do |architecture|
    commands = blocks.fetch("Build #{architecture} Image").dig("task", "jobs", 0, "commands").join("\n")
    archive = "superplane-#{architecture}.oci.tar"
    [archive, "#{archive}.sha256", "#{archive}.descriptor.json"].each do |artifact|
      raise "#{architecture} must replace an existing #{artifact}" unless commands.include?(%(artifact push workflow "#{artifact}" --force))
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
  raise "release prepare must not boot the single-host archive" if blocks.key?("Smoke Test Single Host")
  candidate_verification = blocks.fetch("Verify Release Candidate").dig("task", "jobs", 0, "commands").join("\n")
  raise "candidate verification must check the published prerelease" unless candidate_verification.include?("verify-published-prerelease")
  raise "candidate verification must verify the prepared manifest" unless candidate_verification.include?("verify-manifest")
  raise "candidate verification must export resolved component defaults" unless candidate_verification.include?("component-releases-${RELEASE_CANDIDATE_VERSION}.yaml")
  raise "candidate verification must not exit the Semaphore job" if candidate_verification.match?(/^\s*exit\b/)
  asset_build = blocks.fetch("Build Release Assets").dig("task", "jobs", 0, "commands").join("\n")
  raise "stable assets must use the finalization build" unless asset_build.include?("build-finalized-assets.sh")
  %w[$component_releases $RELEASE_VERSION $SEMAPHORE_GIT_SHA].each do |argument|
    raise "finalization build must receive #{argument}" unless asset_build.include?(argument)
  end

  publish = YAML.safe_load(File.read(ARGV[1]))
  publish_prologue = publish.dig("global_job_config", "prologue", "commands").join("\n")
  unless publish_prologue.include?("verify-superplane-release-branch.sh")
    raise "publication must accept SuperPlane release branches"
  end
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
  finalization = publish.fetch("promotions").find { |promotion| promotion.fetch("name") == "Finalize SuperPlane Release" }
  raise "finalization must reuse the prepare pipeline" unless finalization.fetch("pipeline_file") == "prepare-superplane-release.yml"
  raise "finalization must stay manual" if finalization.key?("auto_promote")
  parameters = finalization.dig("parameters", "env_vars").to_h { |parameter| [parameter.fetch("name"), parameter] }
  candidate = parameters.fetch("RELEASE_CANDIDATE_VERSION")
  raise "finalization must inherit the published candidate" unless candidate.fetch("default_value") == "${{parameters.RELEASE_VERSION}}"
  stable = parameters.fetch("RELEASE_VERSION")
  raise "finalization must require a stable version" unless stable.fetch("required") && stable.fetch("default_value") == "vX.Y.Z"
' "${REPO_ROOT}/.semaphore/prepare-superplane-release.yml" \
  "${REPO_ROOT}/.semaphore/publish-superplane-release.yml"

echo "SuperPlane release pipeline tests passed."
