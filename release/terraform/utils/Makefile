build:
	@docker build --platform linux/amd64 -f Dockerfile -t terraform-utils:latest .

gke.shell: build
	@docker run -it --rm \
	  -v config:/root/.config/gcloud \
	  -v $(shell cd .. && pwd):/workspace \
	  -w /workspace \
		terraform-utils:latest /bin/bash

eks.shell: build
	@docker run -it --rm \
	  -v aws-config:/root/.aws \
	  -v $(shell cd .. && pwd):/workspace \
	  -w /workspace \
		terraform-utils:latest /bin/bash
