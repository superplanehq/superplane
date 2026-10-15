# SuperPlane on Amazon Elastic Kubernetes Service (EKS)

Deploy SuperPlane to EKS with RDS PostgreSQL.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.5.0
- [AWS CLI](https://aws.amazon.com/cli/) installed and configured
- kubectl installed
- AWS account with permissions to create EKS, EFS, RDS, S3, and VPC resources
- A domain that you can configure in DNS

## Deploy

### 1. Select the AWS account

From the repository root, replace `your-aws-profile` with your AWS profile:

```bash
cd release/terraform/eks
export AWS_PROFILE=your-aws-profile
aws sts get-caller-identity
```

Confirm the account ID before you create resources.

### 2. Set your variables

```bash
test -f terraform.tfvars || cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
```

The only required ones are:

```
domain_name       = "superplane.yourdomain.com"
letsencrypt_email = "you@yourdomain.com"
```

You must control the domain so you can add its DNS record after deployment.
If you are deploying outside `us-east-1`, also set `region` and two availability zones in that region:

```
region             = "us-west-2"
availability_zones = ["us-west-2a", "us-west-2b"]
```

Review the defaults before you apply the plan.
Terraform generates a database password if you do not set one.
To use a newer published chart, set `superplane_chart_version` in `terraform.tfvars`.
Leave `superplane_image_tag` empty to use that chart's matching self-hosted image.

### 3. Init, plan and apply

```bash
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

Review the plan before you apply it. A saved plan can contain secrets, and `terraform apply tfplan` does not ask for confirmation.

The deployment takes 15-20 minutes.

### 4. Configure kubectl

```bash
terraform output -raw kubectl_config_command
```

Run the printed command. It uses the configured region and cluster name.

### 5. Configure DNS

After deployment, get the NLB DNS name:

```bash
kubectl get svc -n traefik traefik -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
```

Create a CNAME record in your DNS provider pointing your domain to the NLB DNS name.

For an existing installation, update its CNAME to the new Traefik NLB hostname.
Replacing NGINX removes the old NLB. Plan for an interruption while the NLB and DNS change.

### 6. Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
kubectl get pvc superplane-runner-active-logs -n superplane
```

Access SuperPlane at `https://your-domain.com`

### 7. Install Fleet Manager (optional)

Open **Set up runner fleets** during owner setup. Terraform passes the runner
subnets, security group, instance profile, region, public SuperPlane URL,
cluster name, and Fleet Manager role to SuperPlane. The wizard preloads these
values. Build immutable amd64 and arm64 AMIs with Packer. Set
`runner_image_amd64` and `runner_image_arm64`, and run `terraform apply` again.
The wizard disables fleets whose architecture has no configured AMI.

Terraform creates the AWS runner infrastructure, but it does not install Fleet
Manager. SuperPlane creates a normal personal token when you generate the Helm
heredoc. The standalone chart stores the complete configuration and token in
`fleet-manager-config`. See
[the standalone chart guide](../../fleet-manager-chart/README.md).

## Destroy

Use the same AWS profile and Terraform state as the install.
Confirm the account before you delete resources:

```bash
export AWS_PROFILE=your-aws-profile
export AWS_REGION=us-east-1
aws sts get-caller-identity
terraform state list
kubectl config current-context
```

Set `AWS_REGION` to the value of `region` in `terraform.tfvars`.
Confirm that the kubectl context is the test cluster before you use the recovery commands below.

Before you destroy the infrastructure, disable every fleet in SuperPlane and
wait for active jobs to finish and warm capacity to reach zero. Then stop Fleet
Manager and inspect any runner instances that remain:

```bash
FLEET_MANAGER_NAMESPACE=superplane # Use the namespace from the chart installation.
FLEET_MANAGER_ID=$(helm get values fleet-manager -n "$FLEET_MANAGER_NAMESPACE" --all --output json | jq -er '.manager.id')
RUNNER_INSTANCE_PROFILE_ARN=$(aws iam get-instance-profile \
  --instance-profile-name "$(terraform output -raw runner_instance_profile_name)" \
  --query 'InstanceProfile.Arn' --output text)
helm uninstall fleet-manager -n "$FLEET_MANAGER_NAMESPACE"

aws ec2 describe-instances --region "$AWS_REGION" \
  --filters "Name=tag:superplane_fleet_manager_id,Values=${FLEET_MANAGER_ID}" \
    "Name=iam-instance-profile.arn,Values=${RUNNER_INSTANCE_PROFILE_ARN}" \
    "Name=instance-state-name,Values=pending,running,stopping,stopped" \
  --query 'Reservations[].Instances[].[InstanceId,State.Name,PrivateIpAddress]' \
  --output table
```

Confirm that every listed instance belongs to this installation. Terminate and
wait for any remaining instances before you create the Terraform destroy plan:

```bash
read -r -a RUNNER_INSTANCE_IDS <<< "$(aws ec2 describe-instances --region "$AWS_REGION" \
  --filters "Name=tag:superplane_fleet_manager_id,Values=${FLEET_MANAGER_ID}" \
    "Name=iam-instance-profile.arn,Values=${RUNNER_INSTANCE_PROFILE_ARN}" \
    "Name=instance-state-name,Values=pending,running,stopping,stopped" \
  --query 'Reservations[].Instances[].InstanceId' --output text)"

if ((${#RUNNER_INSTANCE_IDS[@]} > 0)); then
  aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "${RUNNER_INSTANCE_IDS[@]}"
  aws ec2 wait instance-terminated --region "$AWS_REGION" --instance-ids "${RUNNER_INSTANCE_IDS[@]}"
fi
```

If `aws_db_instance.superplane` is in the state, disable its deletion protection in `terraform.tfvars`:

```hcl
rds_deletion_protection = false
```

Apply only this RDS change. Review the plan before you apply it:

```bash
terraform plan -target=aws_db_instance.superplane -out=tfplan
terraform apply tfplan
```

Review and apply a new destroy plan:

```bash
terraform plan -destroy -out=tfplan
terraform apply tfplan
```

The second command deletes resources because `tfplan` contains a destroy plan.
The AWS Load Balancer Controller must delete the Traefik NLB before Terraform removes the NAT gateway and routes.
Do not delete the EKS cluster or remove Terraform state if destroy fails.

### If Traefik or the internet gateway does not delete

Check the Traefik Service and the public IPs in the VPC:

```bash
kubectl describe service traefik -n traefik
aws ec2 describe-network-interfaces --filters Name=vpc-id,Values=<vpc-id> --query 'NetworkInterfaces[?Association.PublicIp!=null].[NetworkInterfaceId,Description,Association.PublicIp]'
```

If the Service shows `service.k8s.aws/resources` and the controller cannot reach AWS, find the Traefik NLB:

```bash
aws elbv2 describe-load-balancers --query 'LoadBalancers[?VpcId==`<vpc-id>`].[LoadBalancerArn,LoadBalancerName]'
aws elbv2 describe-tags --resource-arns <traefik-nlb-arn>
aws elbv2 describe-target-groups --load-balancer-arn <traefik-nlb-arn>
```

Confirm the NLB tag `service.k8s.aws/stack=traefik/traefik` before you delete it.
Record its target group ARNs from the last command. Then remove the NLB and its target groups:

```bash
aws elbv2 delete-load-balancer --load-balancer-arn <traefik-nlb-arn>
aws elbv2 wait load-balancers-deleted --load-balancer-arns <traefik-nlb-arn>
aws elbv2 delete-target-group --target-group-arn <traefik-target-group-arn>
kubectl patch service traefik -n traefik --type merge -p '{"metadata":{"finalizers":[]}}'
terraform destroy
```

Repeat `delete-target-group` for each target group. If the Service is already gone, skip `kubectl patch`.
If the NLB is already gone, skip only the NLB commands. Delete any recorded Traefik target groups before removing the Service finalizer.
If you did not record their ARNs, list target groups and check their tags before you delete them:

```bash
aws elbv2 describe-target-groups --query 'TargetGroups[].[TargetGroupArn,TargetGroupName]'
aws elbv2 describe-tags --resource-arns <target-group-arn>
```

Terraform keeps a final RDS snapshot when it deletes the instance.
The snapshot name has a random suffix.
The suffix changes when an argument that replaces the instance changes.
Terraform also deletes the EFS file system and its active runner logs.
Export any required active logs before you destroy the deployment.
The S3 bucket must be empty before Terraform can delete it. Export any required
blobs, then empty the bucket before you destroy the deployment.

## Notes

- The NLB is created automatically by the AWS Load Balancer Controller
- DNS must be configured as a CNAME pointing to the NLB DNS name
- RDS is deployed in private subnets with no public access
- EKS nodes are in private subnets with NAT gateway for outbound access
- EFS provides shared active-log storage for the Runner API and workers
- S3 provides shared blob storage for the API and runner pods
- EFS mount targets are created in each private subnet
- The EFS access point enforces the UID and GID used by SuperPlane pods
