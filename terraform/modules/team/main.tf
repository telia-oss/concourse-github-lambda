# ------------------------------------------------------------------------------
# Resources
# ------------------------------------------------------------------------------
resource "aws_cloudwatch_event_rule" "main" {
  name                = "concourse-${var.name}-github-${local.config_hash}"
  description         = "Github Lambda team configuration and trigger."
  schedule_expression = "rate(30 minutes)"
  tags                = var.tags
}

resource "aws_cloudwatch_event_target" "main" {
  rule  = aws_cloudwatch_event_rule.main.name
  arn   = var.lambda_arn
  input = local.team_config
}

resource "aws_lambda_permission" "main" {
  statement_id_prefix = "concourse-${var.name}-github-lambda-permission-"
  action              = "lambda:InvokeFunction"
  function_name       = var.lambda_arn
  principal           = "events.amazonaws.com"
  source_arn          = aws_cloudwatch_event_rule.main.arn
}

locals {
  team_config = <<EOF
  {
    "name": "${var.name}",
    "repositories": ${jsonencode(var.repositories)}
  }
EOF
  config_hash = substr(md5(local.team_config), 0, 7)
}
