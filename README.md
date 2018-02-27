## concourse-github-lambda

[![Build Status](https://travis-ci.org/TeliaSoneraNorge/concourse-github-lambda.svg?branch=master)](https://travis-ci.org/TeliaSoneraNorge/concourse-github-lambda)

Lambda function to rotate Github deploy keys used by Concourse teams. See 
the terraform subdirectory for an example that should work (with minimal effort).

### Why?

Our CI/CD (in our case Concourse) needs deploy keys to fetch code from Github.
Instead of having teams do this manually, we can use this Lambda and simply pass
a list of repositories that the team requires access to, and deploy keys will be
generated and written to SSM (where it is available to their pipelines).

### How?

1. This Lambda function is deployed to the same account as our Concourse.
2. It is given a personal access key tied to a machine user.
3. A team adds a CloudWatch event rule with the configuration for which
repositories they need access to.
4. Lambda creates a deploy key and rotates it every 7 days.

### Usage

Be in the root directory:

```bash
make release
```

You should now have a zipped Lambda function. Next, edit [terraform/main.tf](./terraform/main.tf)
to your liking. When done, be in the terraform directory:

```bash
terraform init
terraform apply
```

### Team configuration

Example configuration for a Team (which is then passed as input in the CloudWatch event rule):

```json
{
  "name": "example-team",
  "keyId": "arn:aws:kms:eu-west-1:123456789999:key/fa8eb753-4feb-2c59-b142-03822ca35dbb",
  "repositories": [{
    "concourse-github-lambda"
  }]
}
```

When the function is triggered with the above input, it will create
a deploy key for `TeliaSoneraNorge/concourse-github-lambda` and write
the private key to `/concourse/example-team/concourse-github-lambda-deploy-key`.
