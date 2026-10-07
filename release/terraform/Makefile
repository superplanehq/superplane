# Validate does not call GCP or AWS.
REPO_ROOT := $(abspath ../..)

run = docker run --rm \
	--user $(shell id -u):$(shell id -g) \
	-e HOME=/tmp \
	-e TF_IN_AUTOMATION=1 \
	-v $(REPO_ROOT):/workspace \
	-w /workspace \
	terraform-utils:latest

image:
	$(MAKE) -C utils build

format: image
	$(run) /bin/bash -c 'terraform fmt -recursive release/terraform'

check: image
	$(run) /bin/bash -c 'bash release/terraform/utils/check_terraform.sh'
