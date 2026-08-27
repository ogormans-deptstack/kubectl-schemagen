variable "github_token" {
  type        = string
  sensitive   = true
  description = "GitHub personal access token with repo and admin:org scope"
}

variable "github_owner" {
  type        = string
  default     = "ogormans-deptstack"
  description = "GitHub user or organization that owns the repository"
}

variable "repo_name" {
  type        = string
  default     = "kubectl-schemagen"
  description = "Name of the GitHub repository"
}

variable "tf_encryption_key" {
  type        = string
  sensitive   = true
  description = "Passphrase for OpenTofu state encryption"
}
