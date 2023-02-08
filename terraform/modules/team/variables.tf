# ------------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------------
variable "name" {
  description = "Name of the team (used to give descriptive name to resources)."
  type        = string
}

variable "use_statement_id_prefix" {
  description = "If the name is used as a prefix to a randomised name or not"
  type        = bool
  default     = false
}

variable "lambda_arn" {
  description = "ARN of the Github Lambda."
  type        = string
}

variable "repositories" {
  description = "Valid JSON representation of a Team (see Go code)."
  type        = list(object({ name = string, owner = string, readOnly = bool }))
}

variable "tags" {
  description = "A map of tags (key-value pairs) passed to resources."
  type        = map(string)
  default     = {}
}
