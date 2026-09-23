# SuperPlane on Amazon Elastic Kubernetes Service (EKS)

Deploy SuperPlane to EKS with RDS PostgreSQL.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.5.0
- [AWS CLI](https://aws.amazon.com/cli/) installed and configured
- AWS account with permissions to create EKS, RDS, VPC resources

## Deploy

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

terraform init
terraform apply
```

The deployment takes 15-20 minutes.

## Configure kubectl

```bash
aws eks update-kubeconfig --region us-east-1 --name superplane
```

## Configure DNS

After deployment, get the ALB DNS name:

```bash
kubectl get ingress -n superplane
```

Create a CNAME record in your DNS provider pointing your domain to the ALB DNS name.

## Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
```

Access SuperPlane at `https://your-domain.com`

## Destroy

```bash
# Disable deletion protection on the database
aws rds modify-db-instance \
  --db-instance-identifier superplane-db \
  --no-deletion-protection \
  --apply-immediately

# Wait for modification to complete
aws rds wait db-instance-available --db-instance-identifier superplane-db

terraform destroy
```

## Notes

- The ALB is created automatically by the AWS Load Balancer Controller
- DNS must be configured as a CNAME pointing to the ALB DNS name (not an IP)
- RDS is deployed in private subnets with no public access
- EKS nodes are in private subnets with NAT gateway for outbound access
