# ------------------------------------------------------------------------------
# Resources
# ------------------------------------------------------------------------------
data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

module "lambda" {
  source  = "telia-oss/lambda/aws"
  version = "0.2.0"

  name_prefix = "${var.name_prefix}"
  filename    = "${var.filename}"
  policy      = "${data.aws_iam_policy_document.lambda.json}"
  handler     = "main"
  runtime     = "go1.x"

  environment {
    SECRETS_MANAGER_TOKEN_PATH          = "/${var.secrets_manager_prefix}/{{.Team}}/{{.Owner}}-access-token"
    SECRETS_MANAGER_KEY_PATH            = "/${var.secrets_manager_prefix}/{{.Team}}/{{.Repository}}-deploy-key"
    GITHUB_KEY_TITLE                    = "${var.github_prefix}-{{.Team}}-deploy-key"
    GITHUB_TOKEN_SERVICE_INTEGRATION_ID = "${var.token_service_integration_id}"
    GITHUB_TOKEN_SERVICE_PRIVATE_KEY    = "${var.token_service_private_key}"
    GITHUB_KEY_SERVICE_INTEGRATION_ID   = "${var.key_service_integration_id}"
    GITHUB_KEY_SERVICE_PRIVATE_KEY      = "${var.key_service_private_key}"
  }

  tags = "${var.tags}"
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
