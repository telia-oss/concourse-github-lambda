# ------------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------------
variable "name_prefix" {
  description = "Prefix used for resource names."
}

variable "filename" {
  description = "Path to .zip file containing the handler. (I.e., output of make release)"
}

variable "secrets_manager_prefix" {
  description = "Prefix used for secrets. The Lambda will be allowed to create and write secrets to any secret with this prefix."
  default     = "concourse"
}

variable "github_prefix" {
  description = "Prefix used for Github deploy key name."
  default     = "concourse"
}

variable "github_integration_id" {
  description = "Github App integration ID."
}

variable "github_private_key" {
  description = "Github App private key."
}

variable "tags" {
  description = "A map of tags (key-value pairs) passed to resources."
  type        = "map"
  default     = {}
}
