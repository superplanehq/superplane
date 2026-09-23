build:
	@docker build --platform linux/amd64 -f Dockerfile -t terraform-utils:latest .

shell: build
	@docker run -it --rm \
	  -v config:/root/.config/gcloud \
	  -v aws-config:/root/.aws \
	  -v $(shell cd .. && pwd):/workspace \
	  -w /workspace \
		terraform-utils:latest /bin/bash
