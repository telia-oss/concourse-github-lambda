# ------------------------------------------------------------------------------
# Output
# ------------------------------------------------------------------------------
output "role_arn" {
  value = "${module.lambda.role_arn}"
}

output "role_id" {
  value = "${module.lambda.role_id}"
}

output "function_arn" {
  value = "${module.lambda.arn}"
}

output "function_name" {
  value = "${module.lambda.name}"
}
