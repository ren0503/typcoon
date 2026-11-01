# Terraform Foundation

Minimal Terraform scaffold for future AWS deployment. It declares **no resources** by default and
uses **local state**, so `terraform apply` creates nothing and costs nothing.

## Structure

```text
terraform/
├── versions.tf              # Terraform and provider constraints
├── providers.tf             # AWS provider and default tags
├── variables.tf             # Inputs with validation
├── outputs.tf               # Naming and tag outputs
├── main.tf                  # Naming and tagging locals only
└── environments/
    ├── dev/terraform.tfvars
    └── prod/terraform.tfvars
```

## Usage

```bash
terraform init
terraform validate
terraform plan -var-file=environments/dev/terraform.tfvars
```

`terraform init` downloads the pinned AWS provider. No AWS credentials are needed for `validate`, but
`plan` and `apply` require credentials.

## Conventions

- Name prefix: `<project_name>-<environment>`, for example `tycoon-dev`.
- Common tags: `Project`, `Environment`, `ManagedBy=terraform`, plus `additional_tags`.
- Environment values live in `environments/<env>/terraform.tfvars`.

## Future Work

The scaffold is designed to accommodate VPC, subnets, security groups, EC2 Auto Scaling, an
Application Load Balancer, RDS PostgreSQL, S3, SQS/SNS, Secrets Manager, and CloudWatch. None of
these resources are provisioned now.

A remote state backend (object storage plus locking) is intentionally deferred. Add it before
multiple operators apply changes concurrently.

Kubernetes, EKS, and serverless architectures are explicitly out of scope.
