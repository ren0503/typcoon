terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Local state is intentional for the initial foundation.
  # A remote state backend is documented as future work in README.md.
}
