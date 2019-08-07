terraform {
  required_version = ">= 0.12"
}

provider "aws" {
  version = ">= 2.17"
  region  = var.region
}

module "lambda" {
  source                       = "./modules/lambda"
  name_prefix                  = var.name_prefix
  github_prefix                = "concourse"
  secrets_manager_prefix       = "concourse"
  token_service_integration_id = "sm:///concourse-github-lambda/token-service/integration-id"
  token_service_private_key    = "sm:///concourse-github-lambda/token-service/private-key"
  key_service_integration_id   = "sm:///concourse-github-lambda/key-service/integration-id"
  key_service_private_key      = "sm:///concourse-github-lambda/key-service/private-key"
  tags                         = var.tags
}

resource "aws_iam_role_policy" "secrets" {
  name   = "secrets-policy"
  role   = module.lambda.role_name
  policy = data.aws_iam_policy_document.secrets.json
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "secrets" {
  statement {
    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      "arn:aws:secretsmanager:${var.region}:${data.aws_caller_identity.current.account_id}:secret:/concourse-github-lambda/*/*",
    ]
  }
}

# Each team will need their own Lambda trigger which is CRON triggered
# and passes that teams configuration to the function when it's invoked.
module "team" {
  source     = "./modules/team"
  name       = "${var.name_prefix}-team"
  lambda_arn = module.lambda.arn
  tags       = var.tags

  repositories = [
    {
      name     = "go-hooks"
      owner    = "itsdalmo"
      readOnly = true
    },
  ]
}
