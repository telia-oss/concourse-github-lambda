# ------------------------------------------------------------------------------
# Resources
# ------------------------------------------------------------------------------
data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

locals {
  s3_bucket = var.filename == null && var.s3_bucket == null ? "telia-oss-${data.aws_region.current.name}" : var.s3_bucket
  s3_key    = var.filename == null && var.s3_key == null ? "concourse-github-lambda/v0.9.0.zip" : var.s3_key
}

module "lambda" {
  source  = "telia-oss/lambda/aws"
  version = "3.0.0"

  name_prefix      = var.name_prefix
  filename         = var.filename
  source_code_hash = var.source_code_hash
  s3_bucket        = local.s3_bucket
  s3_key           = local.s3_key
  policy           = data.aws_iam_policy_document.lambda.json
  handler          = "main"
  runtime          = "go1.x"

  environment = {
    SECRETS_MANAGER_TOKEN_PATH          = "/${var.secrets_manager_prefix}/{{.Team}}/{{.Owner}}-access-token"
    SECRETS_MANAGER_KEY_PATH            = "/${var.secrets_manager_prefix}/{{.Team}}/{{.Repository}}-deploy-key"
    GITHUB_KEY_TITLE                    = "${var.github_prefix}-{{.Team}}-deploy-key"
    GITHUB_TOKEN_SERVICE_INTEGRATION_ID = var.token_service_integration_id
    GITHUB_TOKEN_SERVICE_PRIVATE_KEY    = var.token_service_private_key
    GITHUB_KEY_SERVICE_INTEGRATION_ID   = var.key_service_integration_id
    GITHUB_KEY_SERVICE_PRIVATE_KEY      = var.key_service_private_key
  }

  tags = var.tags
}

data "aws_iam_policy_document" "lambda" {
  statement {
    effect = "Allow"

    actions = [
      "ec2:CreateKeyPair",
      "ec2:DeleteKeyPair",
    ]

    resources = [
      "*",
    ]
  }

  statement {
    effect = "Allow"

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = [
      "*",
    ]
  }

  // https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_CreateSecret.html
  // https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_UpdateSecret.html
  statement {
    effect = "Allow"

    actions = [
      "secretsmanager:CreateSecret",
      "secretsmanager:UpdateSecret",
      "secretsmanager:DescribeSecret",
    ]

    resources = [
      "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:/${var.secrets_manager_prefix}/*",
    ]
  }
}

