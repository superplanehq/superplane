#!/usr/bin/env python3
"""Generate the SuperPlane on Azure runbook PDF."""

from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import inch
from reportlab.platypus import (
    ListFlowable,
    ListItem,
    PageBreak,
    Paragraph,
    Preformatted,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)

OUT = Path(__file__).with_name("run-superplane-on-azure.pdf")
DOCS_OUT = Path(__file__).resolve().parents[3] / "docs/contributing/run-superplane-on-azure.pdf"
NAVY = colors.HexColor("#0f2744")
TEAL = colors.HexColor("#1f6f8b")
RULE = colors.HexColor("#d0d7de")
ROW = colors.HexColor("#f4f7fa")


def styles():
    base = getSampleStyleSheet()
    return {
        "cover_kicker": ParagraphStyle(
            "cover_kicker",
            parent=base["Normal"],
            fontName="Helvetica",
            fontSize=10,
            textColor=TEAL,
            spaceAfter=8,
            tracking=1,
        ),
        "cover_title": ParagraphStyle(
            "cover_title",
            parent=base["Title"],
            fontName="Helvetica-Bold",
            fontSize=22,
            leading=26,
            textColor=NAVY,
            alignment=TA_LEFT,
            spaceAfter=10,
        ),
        "cover_sub": ParagraphStyle(
            "cover_sub",
            parent=base["Normal"],
            fontName="Helvetica",
            fontSize=11,
            leading=15,
            textColor=colors.HexColor("#334155"),
            spaceAfter=6,
        ),
        "h1": ParagraphStyle(
            "h1",
            parent=base["Heading1"],
            fontName="Helvetica-Bold",
            fontSize=14,
            leading=18,
            textColor=NAVY,
            spaceBefore=16,
            spaceAfter=8,
        ),
        "h2": ParagraphStyle(
            "h2",
            parent=base["Heading2"],
            fontName="Helvetica-Bold",
            fontSize=12,
            leading=16,
            textColor=TEAL,
            spaceBefore=12,
            spaceAfter=6,
        ),
        "body": ParagraphStyle(
            "body",
            parent=base["Normal"],
            fontName="Helvetica",
            fontSize=10,
            leading=13,
            textColor=colors.HexColor("#1e293b"),
            spaceAfter=6,
        ),
        "code": ParagraphStyle(
            "code",
            parent=base["Code"],
            fontName="Courier",
            fontSize=8,
            leading=11,
            textColor=colors.HexColor("#0f172a"),
            backColor=colors.HexColor("#f8fafc"),
            leftIndent=6,
            rightIndent=6,
            spaceBefore=4,
            spaceAfter=8,
        ),
        "th": ParagraphStyle(
            "th",
            parent=base["Normal"],
            fontName="Helvetica-Bold",
            fontSize=8,
            leading=10,
            textColor=colors.white,
        ),
        "td": ParagraphStyle(
            "td",
            parent=base["Normal"],
            fontName="Helvetica",
            fontSize=8,
            leading=10,
            textColor=colors.HexColor("#1e293b"),
        ),
        "footer": ParagraphStyle(
            "footer",
            parent=base["Normal"],
            fontName="Helvetica",
            fontSize=8,
            textColor=colors.HexColor("#64748b"),
        ),
    }


def bullets(items, s):
    return ListFlowable(
        [ListItem(Paragraph(item, s["body"]), leftIndent=12, bulletColor=TEAL) for item in items],
        bulletType="bullet",
        leftIndent=18,
        spaceAfter=8,
    )


def table(headers, rows, s, col_widths):
    head = [Paragraph(h, s["th"]) for h in headers]
    data = [head]
    for row in rows:
        data.append([Paragraph(c, s["td"]) for c in row])
    t = Table(data, colWidths=col_widths, repeatRows=1)
    t.setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (-1, 0), NAVY),
                ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
                ("BACKGROUND", (0, 1), (-1, -1), colors.white),
                ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.white, ROW]),
                ("GRID", (0, 0), (-1, -1), 0.4, RULE),
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("LEFTPADDING", (0, 0), (-1, -1), 5),
                ("RIGHTPADDING", (0, 0), (-1, -1), 5),
                ("TOPPADDING", (0, 0), (-1, -1), 4),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
            ]
        )
    )
    return t


def header_footer(canvas, doc):
    canvas.saveState()
    canvas.setFillColor(NAVY)
    canvas.rect(0, letter[1] - 28, letter[0], 28, fill=1, stroke=0)
    canvas.setFillColor(colors.white)
    canvas.setFont("Helvetica", 8)
    canvas.drawString(0.75 * inch, letter[1] - 18, "SuperPlane  |  Run on Azure")
    canvas.setFillColor(colors.HexColor("#e2e8f0"))
    canvas.rect(0, 0, letter[0], 32, fill=1, stroke=0)
    canvas.setFillColor(colors.HexColor("#475569"))
    canvas.setFont("Helvetica", 8)
    canvas.drawString(0.75 * inch, 14, "release/terraform/aks")
    canvas.drawRightString(letter[0] - 0.75 * inch, 14, f"Page {doc.page}")
    canvas.restoreState()


def code_block(text, s):
    return Preformatted(text.strip("\n"), s["code"])


def main():
    s = styles()
    doc = SimpleDocTemplate(
        str(OUT),
        pagesize=letter,
        leftMargin=0.75 * inch,
        rightMargin=0.75 * inch,
        topMargin=0.6 * inch,
        bottomMargin=0.55 * inch,
        title="Run SuperPlane on Azure",
        author="SuperPlane",
    )
    story = []
    w = 7.0 * inch

    story.append(Paragraph("INSTALLATION RUNBOOK", s["cover_kicker"]))
    story.append(Paragraph("Run SuperPlane on Azure", s["cover_title"]))
    story.append(
        Paragraph(
            "This document tells you how to build SuperPlane from this repository and install it on Azure Kubernetes Service (AKS).",
            s["cover_sub"],
        )
    )
    story.append(
        Paragraph(
            "The stack uses Azure Blob Storage, optional PostgreSQL Flexible Server, in-cluster RabbitMQ, and nginx with cert-manager.",
            s["cover_sub"],
        )
    )
    story.append(
        Paragraph(
            "Terraform lives in <font face='Courier'>release/terraform/aks</font>. The Helm chart lives in <font face='Courier'>release/superplane-helm-chart/helm</font>. Terraform uses that local chart by default so Azure blob, workload identity, runner API, and Fleet Manager values apply.",
            s["body"],
        )
    )

    story.append(Paragraph("1. What you install", s["h1"]))
    story.append(
        Paragraph(
            "Terraform creates the Azure network, AKS, storage, identities, and the runner gallery. Helm then installs SuperPlane in the cluster.",
            s["body"],
        )
    )
    story.append(
        bullets(
            [
                "AKS with workload identity and Azure CNI Overlay",
                "NAT Gateway for outbound traffic from nodes and runner VMs",
                "Private Azure Blob Storage with user-delegation SAS",
                "Optional PostgreSQL Flexible Server, or Postgres in the cluster",
                "NGINX Ingress Controller with a public load balancer",
                "cert-manager and Let's Encrypt for TLS",
                "Compute Gallery definitions for Azure runner virtual machines",
            ],
            s,
        )
    )

    story.append(Paragraph("2. Prerequisites", s["h1"]))
    story.append(
        bullets(
            [
                "Terraform 1.5.0 or later",
                "Azure CLI, signed in to the target subscription",
                "Docker with Buildx, Helm 3, and kubectl",
                "Packer 1.16.1 or later if you build runner gallery images",
                "A DNS name that you control",
                "A container registry that AKS can pull (GHCR or Azure Container Registry)",
            ],
            s,
        )
    )

    story.append(Paragraph("3. Quota and SKU checks", s["h1"]))
    story.append(
        Paragraph(
            "New Azure subscriptions often have a 4 vCPU regional quota. The default node pool is one Standard_D2s_v4 node (2 vCPUs).",
            s["body"],
        )
    )
    story.append(
        Paragraph(
            "Check Flexible Server SKUs before you apply. If the list is empty, Azure will reject Flexible Server with Version should be in: []. That error is an Azure limit. It is not a SuperPlane version string.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """az postgres flexible-server list-skus --location eastus
az vm list-skus --location eastus --size Standard_D2s_v4 --output table""",
            s,
        )
    )
    story.append(
        Paragraph(
            "If the SKU list is empty, set create_postgresql_flexible_server = false. Do not use Standard_D4s_v5 or Standard_D2ds_v5 unless the subscription lists those sizes.",
            s["body"],
        )
    )

    story.append(Paragraph("4. Build images from this repository", s["h1"]))
    story.append(
        Paragraph(
            "Generated protobuf and OpenAPI files must exist. Run make dev.up and make pb.gen if they are missing.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """cd /path/to/superplane
export IMAGE_TAG="local-$(git rev-parse --short HEAD)"

make image.build IMAGE=superplane IMAGE_TAG="$IMAGE_TAG"

docker build --platform linux/amd64 \\
  -f release/fleet-manager/Dockerfile --target runtime \\
  -t "superplane-fleet-manager:${IMAGE_TAG}" .""",
            s,
        )
    )
    story.append(
        Paragraph(
            "Do not run release/superplane-image/build.sh for this path. That script pushes to GHCR.",
            s["body"],
        )
    )

    story.append(Paragraph("5. Push images to a registry", s["h1"]))
    story.append(
        Paragraph(
            "AKS cannot pull images that exist only on your laptop. Create Azure Container Registry, or push to GHCR if you have access.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """export ACR_NAME="YOUR_ACR_NAME"
az group create --name superplane-images --location eastus
az acr create -g superplane-images -n "$ACR_NAME" --sku Basic -l eastus
az acr login --name "$ACR_NAME"
export ACR_LOGIN="$(az acr show -n "$ACR_NAME" --query loginServer -o tsv)"
export ACR_ID="$(az acr show -n "$ACR_NAME" --query id -o tsv)"

docker tag "superplane:${IMAGE_TAG}" "${ACR_LOGIN}/superplane:${IMAGE_TAG}"
docker tag "superplane-fleet-manager:${IMAGE_TAG}" \\
  "${ACR_LOGIN}/superplane-fleet-manager:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/superplane:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/superplane-fleet-manager:${IMAGE_TAG}" """,
            s,
        )
    )

    story.append(Paragraph("6. Configure Terraform", s["h1"]))
    story.append(
        code_block(
            """cd release/terraform/aks
cp terraform.tfvars.example terraform.tfvars""",
            s,
        )
    )
    story.append(Paragraph("Set at least these values:", s["body"]))
    story.append(
        code_block(
            """subscription_id       = "YOUR_SUBSCRIPTION_ID"
domain_name           = "superplane.example.com"
letsencrypt_email     = "admin@example.com"
location              = "eastus"
node_count            = 1
node_vm_size          = "Standard_D2s_v4"
superplane_image_tag  = "local-YOURSHA"
image_registry        = "YOURACR.azurecr.io"
container_registry_id = "/subscriptions/.../registries/YOURACR"
helm_chart_path       = "../../superplane-helm-chart/helm"
# create_postgresql_flexible_server = false""",
            s,
        )
    )
    story.append(
        Paragraph(
            "Leave fleet_manager_config empty until owner setup is complete. Do not keep a *_override.tf file. Chart path and image registry are Terraform variables.",
            s["body"],
        )
    )

    story.append(Paragraph("7. Apply", s["h1"]))
    story.append(
        code_block(
            """cd release/terraform/aks
terraform init
terraform apply""",
            s,
        )
    )
    story.append(
        Paragraph(
            "The first apply takes 15 to 20 minutes. Terraform assigns AcrPull when container_registry_id is set. You do not need -target for a new install.",
            s["body"],
        )
    )
    story.append(
        Paragraph(
            "If AKS already exists without that role, attach ACR once, then apply again:",
            s["body"],
        )
    )
    story.append(
        code_block(
            'az aks update --resource-group superplane --name superplane --attach-acr "$ACR_NAME"',
            s,
        )
    )

    story.append(Paragraph("8. kubectl, DNS, and first login", s["h1"]))
    story.append(
        Paragraph(
            "kubectl uses localhost:8080 until you load the AKS credentials.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """az aks get-credentials --resource-group superplane --name superplane --admin

kubectl get svc -n ingress-nginx ingress-nginx-controller \\
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}'""",
            s,
        )
    )
    story.append(
        Paragraph(
            "Create a DNS A record for domain_name that points to that IP. Let's Encrypt HTTP-01 needs the public name. A local hosts file does not issue the certificate.",
            s["body"],
        )
    )
    story.append(
        Paragraph(
            "HTTPS on the load balancer IP fails until the certificate is ready. Use port-forward to open the app:",
            s["body"],
        )
    )
    story.append(
        code_block(
            """kubectl port-forward -n superplane svc/superplane-api 8000:8000
# Open http://localhost:8000""",
            s,
        )
    )
    story.append(
        Paragraph(
            "Complete owner setup. Create an admin API token if you will start Fleet Manager.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """kubectl get pods -n superplane
kubectl get certificate -n superplane""",
            s,
        )
    )

    story.append(Paragraph("9. Optional Azure runner VMs", s["h1"]))
    story.append(
        Paragraph(
            "Terraform already creates the gallery, runner subnet, NSG, and identities. Packer publishes Ubuntu images into the gallery. Fleet Manager then creates one VM per runner.",
            s["body"],
        )
    )
    story.append(
        code_block(
            """packer init release/runner/packer/azure/runner.pkr.hcl
packer build -var-file=release/runner/packer/azure/runner.pkrvars.hcl \\
  -var architecture=amd64 -var vm_size=Standard_D2ds_v4 \\
  -var image_name=superplane-runner-amd64 -var image_version=1.0.0 \\
  release/runner/packer/azure/runner.pkr.hcl

release/runner/build.sh "v0.0.0-local" """,
            s,
        )
    )
    story.append(
        Paragraph(
            "Host runner archives on a public HTTPS prefix. The SuperPlane blob account is private. Set fleet_manager_config from terraform output, then apply again.",
            s["body"],
        )
    )

    story.append(Paragraph("10. Destroy", s["h1"]))
    story.append(code_block("cd release/terraform/aks\nterraform destroy", s))
    story.append(
        Paragraph(
            "Flexible Server keeps backups for seven days when that server exists.",
            s["body"],
        )
    )

    story.append(PageBreak())
    story.append(Paragraph("11. Troubleshooting", s["h1"]))
    story.append(
        table(
            ["Symptom", "Action"],
            [
                [
                    "AKS rejects the VM size",
                    "Set node_vm_size to a size from the Azure error list. Default is Standard_D2s_v4.",
                ],
                [
                    "ServiceCidrOverlapExistingSubnetsCidr",
                    "The stack sets aks_service_cidr to 172.16.0.0/16 so it does not overlap 10.0.0.0/16.",
                ],
                [
                    "Insufficient regional vCPU quota",
                    "Set node_count = 1 and Standard_D2s_v4. Raise quota before you grow the pool.",
                ],
                [
                    "kube_admin_config[0] invalid index",
                    "Terraform reads kube_admin_config_raw. Apply again after the cluster exists.",
                ],
                [
                    "Flexible Server Version should be in: []",
                    "Set create_postgresql_flexible_server = false. Helm then runs Postgres in the cluster.",
                ],
                [
                    "Storage 403 Key based authentication",
                    "The account keeps shared keys so Terraform can wait. Pods still use workload identity.",
                ],
                [
                    "Helm labels unmarshal bool",
                    "Workload identity labels use type = string. The chart quotes label values.",
                ],
                [
                    "Helm context deadline exceeded",
                    "The release waits 20 minutes. Check pod logs. Image pulls and migrate can be slow.",
                ],
                [
                    "CrashLoopBackOff Dirty database",
                    "A killed CREATE INDEX CONCURRENTLY leaves schema_migrations dirty. Repair the index, set dirty = false, restart the API.",
                ],
                [
                    "kubectl localhost:8080 refused",
                    "Run az aks get-credentials --resource-group superplane --name superplane --admin.",
                ],
            ],
            s,
            [2.3 * inch, 4.7 * inch],
        )
    )

    story.append(Paragraph("12. Azure resources this stack creates", s["h1"]))
    story.append(
        Paragraph(
            "Defaults use name prefix superplane in eastus. Objects that Azure creates for the ingress load balancer are listed as cloud-provider resources.",
            s["body"],
        )
    )

    story.append(Paragraph("Resource groups", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                ["Resource group", "superplane", "AKS, network, storage, identities, private DNS"],
                ["Resource group", "superplane-runners", "Compute Gallery and runner VM identity"],
            ],
            s,
            [1.5 * inch, 1.8 * inch, 3.7 * inch],
        )
    )

    story.append(Paragraph("Networking", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                ["Virtual network", "superplane-vnet", "Address space 10.0.0.0/16"],
                ["Subnet", "superplane-aks", "AKS nodes, 10.0.0.0/20, Storage service endpoint"],
                ["Subnet", "superplane-postgres", "Flexible Server delegation, 10.0.16.0/24"],
                ["Subnet", "superplane-runners", "Runner VMs, 10.0.17.0/24, no inbound Internet"],
                ["NAT Gateway", "superplane-nat", "Outbound Internet for nodes and runner VMs"],
                ["Public IP", "superplane-nat", "NAT Gateway frontend"],
                ["NSG", "superplane-aks", "AKS subnet"],
                ["NSG", "superplane-runners", "Deny inbound Internet to runner VMs"],
            ],
            s,
            [1.5 * inch, 1.8 * inch, 3.7 * inch],
        )
    )
    story.append(
        Paragraph(
            "AKS Overlay uses Kubernetes service CIDR 172.16.0.0/16 and kube-dns 172.16.0.10.",
            s["body"],
        )
    )

    story.append(Paragraph("Kubernetes on Azure", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                [
                    "AKS cluster",
                    "superplane",
                    "SKU Standard. Workload identity and OIDC issuer enabled. Local admin accounts enabled.",
                ],
                [
                    "Role assignment",
                    "Network Contributor",
                    "AKS cluster identity on the virtual network",
                ],
                [
                    "Role assignment",
                    "AcrPull",
                    "Kubelet identity on ACR when container_registry_id is set",
                ],
                [
                    "Load Balancer (cloud provider)",
                    "created by ingress-nginx",
                    "Public Standard Load Balancer for HTTP and HTTPS",
                ],
                [
                    "Public IP (cloud provider)",
                    "created with the load balancer",
                    "Ingress address. Point DNS A records here.",
                ],
            ],
            s,
            [1.7 * inch, 1.9 * inch, 3.4 * inch],
        )
    )

    story.append(Paragraph("Data", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                ["Storage account", "sp + 8 random chars", "Private blob store. TLS 1.2. GRS."],
                ["Blob container", "superplane", "Platform files. Private access."],
                [
                    "Private DNS zone",
                    "privatelink.postgres.database.azure.com",
                    "Flexible Server private DNS",
                ],
                ["Private DNS VNet link", "superplane-postgres", "Links the zone to the VNet"],
                [
                    "PostgreSQL Flexible Server",
                    "superplane-db",
                    "Created only when create_postgresql_flexible_server is true. Default SKU B_Standard_B2s.",
                ],
                ["PostgreSQL database", "superplane", "Application database"],
                [
                    "PostgreSQL configuration",
                    "require_secure_transport=on",
                    "TLS required for Flexible Server",
                ],
            ],
            s,
            [1.7 * inch, 1.9 * inch, 3.4 * inch],
        )
    )

    story.append(Paragraph("Identities and role assignments", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                [
                    "User-assigned identity",
                    "superplane-app",
                    "Workload identity for API, workers, and websocket",
                ],
                [
                    "User-assigned identity",
                    "superplane-fleet-manager",
                    "Workload identity for Fleet Manager",
                ],
                [
                    "User-assigned identity",
                    "superplane-runner",
                    "Assigned to each runner virtual machine",
                ],
                [
                    "Federated credential",
                    "superplane-app",
                    "system:serviceaccount:superplane:superplane",
                ],
                [
                    "Federated credential",
                    "superplane-fleet-manager",
                    "system:serviceaccount:superplane:superplane-fleet-manager",
                ],
                [
                    "Role",
                    "Storage Blob Data Contributor",
                    "App identity on the storage account",
                ],
                [
                    "Role",
                    "Storage Blob Delegator",
                    "Lets SuperPlane mint user-delegation SAS URLs",
                ],
                [
                    "Role",
                    "Virtual Machine Contributor",
                    "Fleet Manager on resource group superplane-runners",
                ],
                [
                    "Role",
                    "Network Contributor",
                    "Fleet Manager on resource group superplane-runners",
                ],
                ["Role", "Reader", "Fleet Manager on the Compute Gallery"],
                [
                    "Role",
                    "Managed Identity Operator",
                    "Fleet Manager on the runner identity",
                ],
            ],
            s,
            [1.7 * inch, 2.0 * inch, 3.3 * inch],
        )
    )

    story.append(Paragraph("Runner image gallery", s["h2"]))
    story.append(
        table(
            ["Resource", "Default name", "Purpose"],
            [
                ["Azure Compute Gallery", "superplanerunners", "Holds Trusted Launch image definitions"],
                [
                    "Gallery image definition",
                    "superplane-runner-amd64",
                    "linux/amd64 Ubuntu 24.04 runner base",
                ],
                [
                    "Gallery image definition",
                    "superplane-runner-arm64",
                    "linux/arm64 Ubuntu 24.04 runner base",
                ],
            ],
            s,
            [1.8 * inch, 2.0 * inch, 3.2 * inch],
        )
    )
    story.append(
        Paragraph(
            "Packer publishes image versions into those definitions. Terraform does not build the versions.",
            s["body"],
        )
    )

    story.append(Paragraph("13. What runs in the cluster (not Azure PaaS)", s["h1"]))
    story.append(
        bullets(
            [
                "cert-manager and ClusterIssuer letsencrypt-prod",
                "ingress-nginx controller",
                "SuperPlane API, workers, and websocket Deployments",
                "RabbitMQ StatefulSet",
                "Postgres StatefulSet when Flexible Server is off",
                "Fleet Manager Deployment when fleet_manager_config is set",
            ],
            s,
        )
    )

    story.append(Paragraph("14. Created outside Terraform", s["h1"]))
    story.append(
        bullets(
            [
                "Azure Container Registry, if you build images locally",
                "DNS A record for the SuperPlane domain",
                "Packer gallery image versions",
                "Public HTTPS host for runner release archives",
                "Let's Encrypt certificate (cert-manager creates it after DNS works)",
            ],
            s,
        )
    )

    story.append(Paragraph("15. Source paths", s["h1"]))
    story.append(
        table(
            ["Path", "Role"],
            [
                ["release/terraform/aks", "Azure install stack"],
                ["release/superplane-helm-chart/helm", "Helm chart used by this stack"],
                ["Dockerfile --target runner", "SuperPlane application image"],
                ["release/fleet-manager/Dockerfile", "Fleet Manager image"],
                ["release/runner/packer/azure", "Packer gallery images"],
                ["pkg/blob/azure", "Azure Blob provider with workload identity"],
                ["pkg/fleets/provider/azure", "Fleet Manager Azure VM provider"],
            ],
            s,
            [3.0 * inch, 4.0 * inch],
        )
    )

    doc.build(story, onFirstPage=header_footer, onLaterPages=header_footer)
    import shutil

    shutil.copyfile(OUT, DOCS_OUT)
    print(f"Wrote {OUT}")
    print(f"Wrote {DOCS_OUT}")


if __name__ == "__main__":
    main()
