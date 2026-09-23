# Both utils images install the same Terraform. Validate does not call GCP or AWS,
# so the GKE image checks every stack.
REPO_ROOT := $(abspath ../..)

run = docker run --rm \
	--user $(shell id -u):$(shell id -g) \
	-e HOME=/tmp \
	-e TF_IN_AUTOMATION=1 \
	-v $(REPO_ROOT):/workspace \
	-w /workspace \
	gke-image:latest

image:
	$(MAKE) -C utils gke.build

format: image
	$(run) /bin/bash -c 'terraform fmt -recursive release/terraform'

check: image
	$(run) /bin/bash -c 'bash release/terraform/utils/check_terraform.sh'
