gke.build:
	@docker build -f Dockerfile.gke -t gke-image:latest .

gke.shell: gke.build
	@docker run -it --rm \
	  -v config:/root/.config/gcloud \
	  -v $(shell cd .. && pwd):/workspace \
	  -w /workspace \
		gke-image:latest /bin/bash

eks.build:
	@docker build -f Dockerfile.eks -t eks-image:latest .

eks.shell: eks.build
	@docker run -it --rm \
	  --entrypoint /bin/bash \
	  -v aws-config:/root/.aws \
	  -v $(shell cd .. && pwd):/workspace \
	  -w /workspace \
		eks-image:latest