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

### 6. Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
kubectl get pvc superplane-runner-active-logs -n superplane
```

Access SuperPlane at `https://your-domain.com`

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
If the NLB is already gone, skip the NLB and target group commands.

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
