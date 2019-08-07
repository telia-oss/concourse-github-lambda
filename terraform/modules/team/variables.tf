# ------------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------------
variable "name" {
  description = "Name of the team (used to give descriptive name to resources)."
  type        = string
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
