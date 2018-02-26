# ------------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------------
variable "prefix" {
  description = "Prefix used for resource names."
}

variable "zip_file" {
  description = "Path to .zip file containing the handler. (I.e., output of make release)"
}

variable "ssm_prefix" {
  description = "Prefix used for SSM Parameters. The Lambda will be allowed to write to any parameter with this prefix."
  default     = "concourse"
}

variable "github_prefix" {
  description = "Prefix used for Github deploy key name."
  default     = "concourse"
}

variable "github_owner" {
  description = "Owner organization or individual for the repositories."
}

variable "github_token" {
  description = "Access token which grants access to Github API for the repositories."
}

variable "region" {
  description = "Region to use for S3 and SSM clients."
}

variable "tags" {
  description = "A map of tags (key-value pairs) passed to resources."
  type        = "map"
  default     = {}
}
