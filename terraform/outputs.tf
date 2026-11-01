output "name_prefix" {
  description = "Naming prefix for future resources."
  value       = local.name_prefix
}

output "common_tags" {
  description = "Common tags applied through the provider default_tags block."
  value       = local.common_tags
}

output "aws_region" {
  description = "AWS region configured for this environment."
  value       = var.aws_region
}
