provider "aws" {
  region = "eu-west-1"
}

data "aws_region" "current" {}

module "github-lambda" {
  source = "./modules/lambda"

  prefix        = "github-lambda"
  zip_file      = "../concourse-github-lambda.zip"
  ssm_prefix    = "concourse"
  github_prefix = "concourse"
  github_owner  = "itsdalmo"
  github_token  = ""
  region        = "${data.aws_region.current.name}"

  tags {
    environment = "dev"
    terraform   = "True"
  }
}

# Each team will need their own Lambda trigger which is CRON triggered
# and passes that teams configuration to the function when it's invoked.
module "github-lambda-trigger" {
  source = "./modules/trigger"

  prefix = "example-team"
  lambda_arn = "${module.github-lambda.function_arn}"
  team_config = <<EOF
{
  "name": "example-team",
  "keyId": "",
  "repositories": [
    {
      "name": "go-hooks",
      "readOnly": "true"
    }
  ]
}
EOF
}

output "lambda_arn" {
  value = "${module.github-lambda.function_arn}"
}
