provider "aws" {
  region = "eu-west-1"
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

module "github-lambda" {
  source = "./modules/lambda"

  name_prefix            = "github-lambda"
  filename               = "../concourse-github-lambda.zip"
  github_prefix          = "concourse"
  secrets_manager_prefix = "concourse"
  github_integration_id  = "sm:///concourse-github-lambda/github/integration-id"
  github_private_key     = "sm:///concourse-github-lambda/github/private-key"

  tags {
    environment = "dev"
    terraform   = "True"
  }
}

resource "aws_iam_role_policy" "secrets" {
  name   = "github-lambda-secrets-policy"
  role   = "${module.github-lambda.role_name}"
  policy = "${data.aws_iam_policy_document.secrets.json}"
}

data "aws_iam_policy_document" "secrets" {
  statement {
    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:/concourse-github-lambda/github/*",
    ]
  }
}

# Each team will need their own Lambda trigger which is CRON triggered
# and passes that teams configuration to the function when it's invoked.
module "github-lambda-trigger" {
  source = "./modules/trigger"

  name_prefix = "example-team"
  lambda_arn  = "${module.github-lambda.function_arn}"

  team_config = <<EOF
{
  "name": "example-team",
  "repositories": [
    {
      "name": "go-hooks",
      "owner": "itsdalmo",
      "readOnly": "true"
    }
  ]
}
EOF
}

output "lambda_arn" {
  value = "${module.github-lambda.function_arn}"
}
