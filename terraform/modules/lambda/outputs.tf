# ------------------------------------------------------------------------------
# Output
# ------------------------------------------------------------------------------
output "role_arn" {
  value = "${module.lambda.role_arn}"
}

output "function_arn" {
  value = "${module.lambda.arn}"
}

output "function_name" {
  value = "${module.lambda.name}"
}
