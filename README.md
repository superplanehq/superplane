# SuperPlane

SuperPlane is an AI-native DevOps control plane. Our mission is to build the
platform teams use to ship and manage software in the AI era.

Agents are helping us write an order of magnitude more code, while systems have
become too complex for human-driven ops alone. We're rethinking DevOps from
first principles for the AI era: a single control layer where engineers and
agents safely collaborate.

## Key Capabilities

- **AI-Native Architecture** - Built from the ground up for the AI era, enabling seamless collaboration between engineers and AI agents to manage increasingly complex DevOps systems
- **Cross-Platform Workflow Orchestration** - Connect and coordinate workflows across multiple DevOps tools, platforms, and services from a single interface
- **Event-Driven Automation** - Build workflows that automatically respond to code pushes, deployments, alerts, and custom triggers
- **Visual Workflow Builder** - Design and manage complex DevOps processes with an intuitive visual interface and real-time status updates
- **Operational Knowledge Centralization** - Create living documentation of your DevOps processes that's easy to understand and maintain

## Quick Start

The fastest way to try SuperPlane is to run the latest version of the SuperPlane
Docker container on your own machine. You'll have a working SuperPlane instance
in less than a minute, without provisioning any cloud infrastructure.

```
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

## Production Installation

For a permanent, production-ready installation, SuperPlane can be deployed on a
single host or on Kubernetes. The single-host installation is ideal for smaller
deployments and provides automatic SSL certificate management. Kubernetes
deployment offers better scalability and high availability for larger teams.

- **[Single Host Installation](https://docs.superplane.com/installation/single-host/aws-ec2/)** - Deploy on AWS EC2, GCP Compute Engine, or other cloud providers
- **[Kubernetes Installation](https://docs.superplane.com/installation/kubernetes/gke/)** - Deploy on GKE, EKS, or any Kubernetes cluster

## Contributing

Found a bug or have a feature idea? Check our **[Contributing Guide](CONTRIBUTING.md)** to get started.

## Get In Contact

- **[Discord](https://discord.gg/KC78eCNsnw)** - Join our community for discussions, questions, and collaboration
- **[X](https://x.com/superplanehq)** - Follow us for updates and announcements
