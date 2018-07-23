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

variable "token_service_integration_id" {
  description = "Integration ID for the access token Github App."
}

variable "token_service_private_key" {
  description = "Private key for the access token Github App."
}

variable "key_service_integration_id" {
  description = "Integration ID for the deploy key Github App."
}

variable "key_service_private_key" {
  description = "Private key for the deploy key Github App."
}

variable "tags" {
  description = "A map of tags (key-value pairs) passed to resources."
  type        = "map"
  default     = {}
}
