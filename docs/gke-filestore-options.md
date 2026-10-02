# GKE Filestore Options for Active Runner Logs

> Temporary infrastructure planning note. Do not commit this file.
>
> Prices and cluster facts were checked on October 1, 2026.

## Recommendation

Use one NFSv4.1 Filestore share for active runner logs.
Mount the same `ReadWriteMany` volume in all runner API and log archiver pods.

Evaluate the options in this order:

1. Use a 100 GiB Regional instance if the projects have small-capacity access.
2. Use a 1 TiB Zonal instance if small Regional instances are unavailable.
3. Use a 1 TiB Regional instance if regional resilience is required.
4. Use Enterprise single-share only if its GKE integration provides a required benefit.

Google recommends the Regional tier for single-share workloads.
Enterprise primarily targets GKE multishare workloads.
Enterprise multishare does not support NFSv4.1, so it does not fit this design.

## Current GKE State

Production and staging currently use:

| Setting | Value |
| --- | --- |
| Cluster name | `superplane` |
| Region | `us-east4` |
| GKE version | `1.35.8-gke.1036000` |
| Production project | `superplane-production` |
| Staging project | `superplane-stg` |
| Managed Filestore CSI driver | Disabled |
| Filestore API | Disabled |

GKE 1.35 supports Filestore NFSv4.1.
Google requires GKE 1.33 or later for the applicable NFSv4.1 CSI configurations.

Before provisioning storage:

1. Enable `file.googleapis.com` in both projects.
2. Enable the managed Filestore CSI driver in both clusters.
3. Confirm small-capacity Regional access in staging.
4. Confirm the selected tier and capacity have quota in `us-east4`.

## Required Storage Behavior

The active log filesystem requires:

- `ReadWriteMany` access.
- Concurrent mounts from different GKE nodes.
- POSIX-compatible file operations.
- Cross-client advisory file locks.
- Atomic file replacement through `rename`.
- Durable `fsync` behavior.

Filestore provides these features with NFSv4.1.
Specify `NFS_V4_1` explicitly because the default can be NFSv3.

NFSv4.1 uses server-managed lease locks.
The server releases a lock when a failed client stops renewing its lease.
This behavior is safer than NFSv3 client-managed NLM locks.

## Sharing One Volume Across Deployments

One `ReadWriteMany` PVC can be mounted by multiple pods and Deployments.
The PVC consumers must be in the same Kubernetes namespace.

Mount the volume in:

- Every runner API pod.
- Every pod that runs the runner task log archiver.
- Every pod that runs active log expiration or cleanup.

Use the same mount path and environment variables in all consumers.

If consumers use different namespaces, they cannot reference one PVC directly.
Static PVs can expose the same Filestore share through separate PVCs.
Keeping all consumers in one namespace is simpler.

## Filestore Options

The estimates below use 730 hours per month and `us-east4` list prices.
They exclude discounts, backups, snapshots, and network charges.

| Option | Capacity | Availability | Approximate cost/month | Nominal performance |
| --- | ---: | --- | ---: | --- |
| Regional, custom performance | 100 GiB | Regional | $115 | 2,000 read IOPS, 600 write IOPS, 47/16 MiB/s |
| Zonal, custom performance | 1 TiB | Zonal | $201 | 4,000 read IOPS, 1,200 write IOPS, 94/32 MiB/s |
| Zonal, standard performance | 1 TiB | Zonal | $256 | 9,200 read IOPS, 2,600 write IOPS, 260/88 MiB/s |
| Regional, custom performance | 1 TiB | Regional | $363 | 4,000 read IOPS, 1,200 write IOPS, 94/32 MiB/s |
| Regional, standard performance | 1 TiB | Regional | $461 | 12,000 read IOPS, 4,000 write IOPS, 120/100 MiB/s |
| Enterprise single-share | 1 TiB | Regional | $461 | 12,000 read IOPS, 4,000 write IOPS, 120/100 MiB/s |

The custom-performance rows use the minimum applicable provisioned IOPS.
Confirm the values in the Google Cloud pricing calculator before deployment.

### Small Regional

This is the preferred starting option if it is available.

- Minimum capacity: 100 GiB.
- Minimum provisioned performance: 2,000 IOPS.
- Capacity can change in 1 GiB increments.
- NFSv4.1 is supported.
- Availability is regional.
- Access to capacities below 1 TiB can require project eligibility.

At the 10 MiB task log limit, 100 GiB holds approximately 10,000 maximum-size
active logs before filesystem overhead and operating headroom.
Use a lower operational threshold.

### Zonal

This is the lower-cost fallback if small Regional is unavailable.

- Minimum capacity: 1 TiB.
- Capacity changes in 256 GiB increments below 10 TiB.
- NFSv4.1 is supported.
- Availability is zonal.
- Pods in other zones can mount the share through the network.

A zone outage can make active logs unavailable.
Choose the Filestore zone with the cluster topology and network costs in mind.

### Regional

Use Regional when active log availability must survive a zone outage.

- Minimum capacity: 100 GiB with small-capacity access.
- Otherwise, the minimum capacity is 1 TiB.
- NFSv4.1 is supported.
- Custom performance can reduce the 1 TiB starting price.
- Google recommends this tier for non-multishare regional workloads.

### Enterprise

Enterprise single-share is valid but starts at 1 TiB.
It has regional availability and NFSv4.1 support.

Enterprise multishare permits smaller PVCs, but the underlying pool still
starts at 1 TiB.
NFSv4.1 is not supported for Enterprise multishare.
Do not select multishare for the active log store.

## Performance and Lock Limits

Filestore performance values are raw filesystem limits.
They are not equal to active log append requests per second.

One active log append includes:

- An advisory lock operation.
- Manifest metadata reads.
- A data append.
- A data `fsync`.
- A temporary manifest write.
- A manifest `fsync`.
- An atomic rename.
- A directory `fsync`.

Benchmark the complete append path from multiple GKE nodes.
Google recommends multiple clients and `nconnect=2` for these service tiers.

Filestore documents these lock limits for Zonal, Regional, and Enterprise:

- Maximum 250 locks on one file.
- Recommended maximum 250 file-locking clients per instance.

A file-locking client is an NFS client with an acquired or requested lock.
For GKE, the relevant client count is approximately the number of participating
nodes, not the number of tasks.

The active log store uses 4,096 lock shard files.
Each operation holds one shard lock for a short period.
The per-file limit is unlikely to be the first constraint.
Monitor lock wait time and the number of GKE nodes that mount the share.

## Provisioning Choice

Two provisioning models are possible.

### Terraform-managed Filestore

Terraform creates:

- The Filestore instance and share.
- Network configuration.
- Deletion protection.
- Optional backup policy.
- Outputs for location, instance name, share name, and IP address.

Helm creates a static PV and PVC from those outputs.

This model gives infrastructure code explicit control over storage deletion and
migrations.
It is the safer production default.

Use `Retain` as the PV reclaim policy.
Deleting a Helm release must not delete the Filestore instance.

### CSI dynamic provisioning

Terraform enables the API and cluster CSI add-on.
Helm creates a `StorageClass` and PVC.
The CSI driver creates the Filestore instance.

This model uses less infrastructure code.
However, PVC lifecycle and Filestore lifecycle become more closely connected.
Review the reclaim policy before using this model in production.

Small-capacity Regional support can differ between Filestore and CSI workflows.
Use a Terraform-managed instance with a static PV if dynamic provisioning
cannot create the selected option.

## Kubernetes Configuration Shape

For dynamic provisioning, the StorageClass must use:

```yaml
provisioner: filestore.csi.storage.gke.io
parameters:
  protocol: NFS_V4_1
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
mountOptions:
  - nconnect=2
```

Add the selected `tier` and network parameters.
Do not set `multishare: "true"` because it prevents NFSv4.1 use.

For static provisioning, the PV uses this shape:

```yaml
spec:
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  volumeMode: Filesystem
  mountOptions:
    - nconnect=2
  csi:
    driver: filestore.csi.storage.gke.io
    volumeHandle: modeInstance/<location>/<instance>/<share>
    volumeAttributes:
      ip: <filestore-ip>
      volume: <share>
      protocol: NFS_V4_1
```

The PVC also requests `ReadWriteMany`.

Mount the claim in each consumer:

```yaml
volumes:
  - name: runner-active-logs
    persistentVolumeClaim:
      claimName: runner-active-logs

volumeMounts:
  - name: runner-active-logs
    mountPath: /var/lib/superplane/active-runner-logs
```

Configure SuperPlane:

```yaml
RUNNER_ACTIVE_LOG_FS_PATH: /var/lib/superplane/active-runner-logs
```

Do not configure fallback paths for the initial deployment.

## Zonal-to-Regional or Enterprise Migration

Filestore does not support an in-place tier change.
The source and destination are separate instances with different NFS endpoints.

Active logs do not require a backup, restore, or bulk copy.
The filesystem store supports one primary path and fallback paths.

Use this migration:

1. Provision the destination Filestore instance and PVC.
2. Mount both the source and destination PVCs in every consumer.
3. Deploy with the source as primary and the destination as a fallback.
4. Confirm that all replicas can read and lock both paths.
5. Change the destination to primary and keep the source as a fallback.
6. Complete the rolling deployment.
7. Let tasks on the source path finish and archive normally.
8. Wait until active log cleanup removes all source task directories.
9. Remove the source fallback path and mount.
10. Delete the source instance after the rollback period.

Example cutover configuration:

```yaml
RUNNER_ACTIVE_LOG_FS_PATH: /var/lib/superplane/active-runner-logs-new
RUNNER_ACTIVE_LOG_FS_FALLBACK_PATHS: /var/lib/superplane/active-runner-logs-old
```

New tasks use the primary path.
Existing tasks remain on the path selected during initialization.
Long-running tasks do not block new tasks during the migration.
All replicas can access both task groups.

During a rolling update, old replicas can still initialize tasks on the old
primary path.
New replicas find those tasks through the fallback path.
Remove the old path only after the rollout and old-path cleanup complete.

Google also supports backup and restore between compatible tiers.
The destination must use the same NFS protocol and sufficient capacity.
That flow is unnecessary for ephemeral active logs.

## Terraform Checklist

- [ ] Enable `file.googleapis.com`.
- [ ] Enable the GKE Filestore CSI driver.
- [ ] Confirm `us-east4` tier availability and quota.
- [ ] Confirm small-capacity Regional eligibility.
- [ ] Select `NFS_V4_1`.
- [ ] Place the instance on the GKE VPC.
- [ ] Permit NFSv4.1 traffic on TCP port 2049.
- [ ] Configure deletion protection.
- [ ] Select `Retain` for the Kubernetes PV.
- [ ] Export values required by the Helm release.
- [ ] Add capacity, IOPS, latency, and availability alerts.

## Helm Checklist

- [ ] Create or reference one `ReadWriteMany` PVC.
- [ ] Mount the PVC in all runner API replicas.
- [ ] Mount the PVC in all log archiver and cleanup replicas.
- [ ] Use the same paths and environment variables in every replica.
- [ ] Set `RUNNER_ACTIVE_LOG_FS_PATH`.
- [ ] Support a list of fallback PVC mounts and paths.
- [ ] Verify container UID, GID, and directory permissions.
- [ ] Confirm that a Helm uninstall cannot delete the Filestore instance.
- [ ] Add a migration mode that mounts old and new PVCs together.

## Staging Validation

Before production:

1. Create the selected option in staging.
2. Run runner API and archiver replicas on different GKE nodes.
3. Confirm that all pods mount the same share.
4. Run concurrent append, read, archive, and cleanup operations.
5. Delete a pod while it holds a lock.
6. Confirm that NFSv4.1 lease recovery releases the lock.
7. Benchmark 64 KiB and 128 KiB appends.
8. Record append latency, lock wait time, IOPS, and throughput.
9. Test a rolling primary and fallback path migration.
10. Confirm that completed logs remain available from blob storage.

## Sources

- [GKE Filestore CSI driver](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/filestore-csi-driver)
- [Filestore NFSv4.1](https://cloud.google.com/filestore/docs/configure-nfsv4)
- [Filestore service tiers](https://cloud.google.com/filestore/docs/service-tiers)
- [Filestore Zonal tier](https://cloud.google.com/filestore/docs/service-tiers-zonal)
- [Filestore Regional tier](https://cloud.google.com/filestore/docs/service-tiers-regional)
- [Filestore Enterprise tier](https://cloud.google.com/filestore/docs/service-tiers-enterprise)
- [Create a Filestore instance](https://cloud.google.com/filestore/docs/creating-instances)
- [Filestore performance](https://cloud.google.com/filestore/docs/performance)
- [Filestore limits](https://cloud.google.com/filestore/docs/limits)
- [Filestore pricing](https://cloud.google.com/filestore/pricing)
