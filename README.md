## concourse-github-lambda

[![Build Status](https://travis-ci.org/telia-oss/concourse-github-lambda.svg?branch=master)](https://travis-ci.org/telia-oss/concourse-github-lambda)

Lambda function to rotate Github deploy keys used by Concourse teams. See
the terraform subdirectory for an example that should work (with minimal effort).

### Why?

Our CI/CD (in our case Concourse) needs deploy keys to fetch code from Github.
Instead of having teams do this manually, we can use this Lambda and simply pass
a list of repositories that the team requires access to, and deploy keys will be
generated and written to Secrets Manager (where it is available to their pipelines).

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

You should now have a zipped Lambda function. Next, edit [terraform/example.tf](./terraform/example.tf)
to your liking. When done, be in the terraform directory:

```bash
terraform init
terraform apply
```

NOTE: The `aws/secretsmanager` KMS Key Alias has to be created/exist before the lambda is deployed.

### Team configuration

Example configuration for a Team (which is then passed as input in the CloudWatch event rule):

```json
{
  "name": "example-team",
  "repositories": [
    {
      "name": "concourse-github-lambda",
      "owner": "telia-oss",
      "readOnly": "true"
    }
  ]
}
```

When the function is triggered with the above input, it will create
a deploy key for `telia-oss/concourse-github-lambda` and write
the private key to `/concourse/example-team/concourse-github-lambda-deploy-key`.

### Required secrets 

We recommend using secrets manager or SSM (over KMS). See below for an example of 
setting up the required secrets using Secrets Manager:

```bash
aws secretsmanager create-secret \
  --name /concourse-github-lambda/token-service/integration-id \
  --secret-string "13024" \
  --region eu-west-1

aws secretsmanager create-secret \
  --name /concourse-github-lambda/token-service/private-key \
  --secret-string file:///Users/someone/Downloads/concourse-github-token-service.pem \
  --region eu-west-1

aws secretsmanager create-secret \
  --name /concourse-github-lambda/key-service/integration-id \
  --secret-string "13025" \
  --region eu-west-1

aws secretsmanager create-secret \
  --name /concourse-github-lambda/key-service/private-key \
  --secret-string file:///Users/someone/Downloads/concourse-github-key-service.pem \
  --region eu-west-1
```

To update the values, use `update-secret` and `--secret-id` instead of `create-secret` and `--name`.
Otherwise the arguments can remain the same.
