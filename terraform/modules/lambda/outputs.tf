# ------------------------------------------------------------------------------
# Output
# ------------------------------------------------------------------------------
output "role_arn" {
  value = "${module.lambda.role_arn}"
}

output "role_name" {
  value = "${module.lambda.role_name}"
}

output "function_arn" {
  value = "${module.lambda.arn}"
}

output "function_name" {
  value = "${module.lambda.name}"
}
