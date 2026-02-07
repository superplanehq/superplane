# Proposed Issue: [Google Cloud] Base

## Description

Create a SuperPlane integration for Google Cloud (GCP) and two starter components (one trigger, one action) so users can build workflows across Google Cloud services in SuperPlane. This proposal mirrors the Azure base issue and establishes the GCP base integration and the first components listed below.

Link: https://cloud.google.com/

## Connection Method

Determine the best Google Cloud authentication strategy for this base (for example: Workload Identity Federation with OIDC, service account key, or managed identity for GCE/GKE) and document the chosen approach. Look at the AWS integration and choose the best approximate for GCP.

IMPORTANT: Before implementation begins, the assignee must research GCP auth options and verify the proposed approach with the SuperPlane team.

Suggested starting point: Workload Identity Federation (OIDC) with a dedicated service account and least-privilege IAM roles, using the Google Cloud Security Token Service (STS) to exchange the SuperPlane OIDC token for a short-lived access token. Store the service account email, workload identity pool/provider resource name, and any required audience/issuer settings in the integration configuration. Avoid long-lived service account JSON keys unless there is no feasible alternative.

## First Components

IMPORTANT: Creating at least 1 trigger and 1 action component is required for completing this issue.

**1/ On VM Created (Trigger)**

Emits when a new Compute Engine VM is created (provisioning succeeded). Trigger should use the appropriate Google Cloud event source (Eventarc + Cloud Audit Logs or Cloud Logging sink to Pub/Sub) and emit the VM creation payload to start SuperPlane workflow executions.

**2/ Create Virtual Machine (Action)**

Provision a new Compute Engine VM and return instance ID, self link, internal/external IPs, status, and key metadata.

Requirement: this action must expose all options available in the Google Cloud Console "Create an instance" UI.

Examples of covered areas include:

- Basics: project, region/zone, instance name, machine family/series, machine type, VM provisioning model (standard/spot), availability policy, boot disk image
- Identity and API access: service account, OAuth scopes, OS Login, IAM settings for guest OS access
- Security: Shielded VM (secure boot, vTPM, integrity monitoring), Confidential VM
- Disks: boot disk type/size/encryption, additional persistent disks, local SSD, delete-on-termination
- Networking: VPC, subnet, NIC type, internal/external IP, static IP, network tags, firewall rules, IP stack type
- Management: metadata, startup script, automation, maintenance policy, automatic restart, host maintenance
- Advanced: GPU accelerators, placement policy, sole-tenant/host affinity, resource policies
- Labels

## Acceptance Criteria (for SuperPlane team)

- [ ] integration works
- [ ] starter components work
- [ ] has proper tests
- [ ] has proper documentation
- [ ] passed code quality review
- [ ] passed functionality review
- [ ] passed ui/ux review

## Potential Components

Once the Base Integration is completed, the SuperPlane team will choose from the components below. The base integration should be built in a way that does not block their creation.

[Cloud Monitoring API] Run Query
[Cloud Monitoring API] Get Metrics
[Cloud Monitoring API] Get Metric Descriptors
[Cloud Monitoring API] Create/Update Alert Policy
[Cloud Monitoring API] On Alert
[Cloud Logging API] List Log Entries
[Cloud Logging API] Create Sink
[Cloud Logging API] On Log Entry (via Sink)
[Cloud Storage API] Upload Object
[Cloud Storage API] Download / Get Object
[Cloud Storage API] List Objects
[Cloud Storage API] List Buckets
[Cloud Storage API] Get Object Metadata
[Cloud Storage API] Delete Object
[Cloud Storage API] On Object Finalized
[Pub/Sub API] Publish Message
[Pub/Sub API] Pull and Ack Message
[Pub/Sub API] On Message
[Pub/Sub API] List Topics / Subscriptions
[Pub/Sub API] Create Topic
[Pub/Sub API] Create Subscription
[Compute Engine API] Start/Stop VM
[Compute Engine API] Get VM
[Compute Engine API] List VMs
[Compute Engine API] Resize VM
[Compute Engine API] Create Snapshot
[Cloud SQL Admin API] List Instances
[Cloud SQL Admin API] Get Instance
[Cloud SQL Admin API] List Databases
[Cloud SQL Admin API] Create Database
[GKE API] List Clusters
[GKE API] Get Cluster
[GKE API] Scale Node Pool
[Cloud Run API] Deploy Service
[Cloud Run API] Update Service
[Cloud Run API] List Services
[Cloud Run API] Get Service
[Artifact Registry API] On Image Push
[Artifact Registry API] List Repositories
[Artifact Registry API] List Packages
[Artifact Registry API] List Tags
[Cloud Build API] Run Build
[Cloud Build API] On Build Finished
[Cloud Build API] List Triggers
[Secret Manager API] Get Secret
[Secret Manager API] Add Secret Version
[Secret Manager API] List Secrets
[Secret Manager API] Disable Secret Version
[IAM Credentials API] Generate Access Token
[Security Token Service] Exchange OIDC Token
