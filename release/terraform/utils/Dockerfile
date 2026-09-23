FROM alpine/helm:4.3.0@sha256:a6cf54599ccb99d90cf0712b30f03fdb3cab062e6b94e0418cc4db7e8a1464b2

# The Helm image entrypoint is helm. This image is a shell.
ENTRYPOINT []

ENV PATH="/google-cloud-sdk/bin:${PATH}" \
    CLOUDSDK_PYTHON=python3 \
    CLOUDSDK_PYTHON_SITEPACKAGES=1

RUN apk add --no-cache \
      aws-cli \
      bash \
      curl \
      gzip \
      jq \
      libc6-compat \
      openssl \
      py3-crcmod \
      py3-openssl \
      python3 \
      tar \
      unzip \
      vim \
    && curl -fsSL -o /tmp/google-cloud-cli.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-586.0.0-linux-x86_64.tar.gz \
    && printf '%s  /tmp/google-cloud-cli.tar.gz\n' '6c774c76793eedd501150b59da653610fbe3eaac169e822965b722de75a2f001' > /tmp/google-cloud-cli.sha256 \
    && sha256sum -c /tmp/google-cloud-cli.sha256 \
    && tar -xzf /tmp/google-cloud-cli.tar.gz -C / \
    && rm /tmp/google-cloud-cli.tar.gz /tmp/google-cloud-cli.sha256 \
    && rm -rf /google-cloud-sdk/platform/bundledpythonunix \
    && gcloud config set core/disable_usage_reporting true \
    && gcloud config set component_manager/disable_update_check true \
    && gcloud components install gke-gcloud-auth-plugin --quiet \
    && curl -fsSL -o /usr/local/bin/kubectl https://dl.k8s.io/release/v1.36.5/bin/linux/amd64/kubectl \
    && printf '%s  /usr/local/bin/kubectl\n' '33bc88a24c3b09cf55bfd59e6ca977881b61fc3f25ab65bc80c65b430248648e' > /tmp/kubectl.sha256 \
    && sha256sum -c /tmp/kubectl.sha256 \
    && rm /tmp/kubectl.sha256 \
    && chmod +x /usr/local/bin/kubectl \
    && curl -fsSL -o /tmp/terraform.zip https://releases.hashicorp.com/terraform/1.16.4/terraform_1.16.4_linux_amd64.zip \
    && printf '%s  /tmp/terraform.zip\n' 'dc94af0eef1147718ad7c8daea792ed199e3e0492eec180d0adafa2a65a879df' > /tmp/terraform.sha256 \
    && sha256sum -c /tmp/terraform.sha256 \
    && unzip /tmp/terraform.zip -d /usr/local/bin \
    && rm /tmp/terraform.zip /tmp/terraform.sha256 /usr/local/bin/LICENSE.txt \
    && chmod +x /usr/local/bin/terraform \
    && curl -fsSL -o /tmp/eksctl.tar.gz https://github.com/eksctl-io/eksctl/releases/download/v0.230.0/eksctl_Linux_amd64.tar.gz \
    && printf '%s  /tmp/eksctl.tar.gz\n' 'a2060956f117c3065abafda5c1f681679b9c3716675d70ce4ffff46033b02c35' > /tmp/eksctl.sha256 \
    && sha256sum -c /tmp/eksctl.sha256 \
    && tar -xzf /tmp/eksctl.tar.gz -C /usr/local/bin \
    && rm /tmp/eksctl.tar.gz /tmp/eksctl.sha256 \
    && helm version --short \
    && terraform version \
    && kubectl version --client=true \
    && gcloud version \
    && gke-gcloud-auth-plugin --version \
    && aws --version \
    && eksctl version
