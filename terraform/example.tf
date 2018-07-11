provider "aws" {
  region = "eu-west-1"
}

module "github-lambda" {
  source = "./modules/lambda"

  name_prefix            = "github-lambda"
  filename               = "../concourse-github-lambda.zip"
  github_prefix          = "concourse"
  github_token           = ""
  secrets_manager_prefix = "concourse"

  tags {
    environment = "dev"
    terraform   = "True"
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
