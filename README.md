## concourse-github-lambda

[![Build Status](https://travis-ci.org/telia-oss/concourse-github-lambda.svg?branch=master)](https://travis-ci.org/telia-oss/concourse-github-lambda)

Lambda function for handling Github access tokens and deploy keys used by Concourse teams. See
the terraform subdirectory for an example that should work (with minimal effort).

### Why?

Our CI/CD (in our case Concourse) needs deploy keys to fetch code from Github, and
access tokens to set statuses on commits or comment on pull requests.
Instead of having teams do this manually, we can use this Lambda and simply pass
a list of repositories that the team requires access to, and deploy keys will be
generated and written to Secrets Manager (where it is available to their pipelines).

### How?

1. This Lambda function is deployed to the same account as our Concourse.
2. It is given an integration ID and private key for two separate [Github Apps](https://developer.github.com/apps/).
3. A team adds a CloudWatch event rule with the configuration for which repositories they need access to, and under which 
organisation. 
4. The lambda creates/rotates an access token and deploy key for each team, every 30min and 7 days respectively.

### Usage

After you have checked out the [prerequisites](#prerequisites), either download a zip from the 
[releases](https://github.com/telia-oss/concourse-github-lambda/releases), or build it yourself by 
running `make release` in the root of this repository. After you have a binary, you can edit 
[terraform/example.tf](./terraform/example.tf) to your liking and deploy the example by running:

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

When the function is triggered with the above input, it will create a deploy key for `telia-oss/concourse-github-lambda`,
write a private key to `/concourse/example-team/concourse-github-lambda-deploy-key` and access token to 
`/concourse/example-team/telia-oss-access-token`.

### Prerequisites

#### Github Apps


This Lambda requires credentials for two separate Github Apps in order to generate deploy keys and access tokens. See the 
official documentation on [Creating a Github App](https://developer.github.com/apps/building-github-apps/creating-a-github-app/),
and grant them the following permissions:

- key-service (generates deploy keys): [Repository administration (`write`)](https://developer.github.com/v3/apps/permissions/#permission-on-administration)
- token-service (generates access tokens): ... any permissions really, or no permissions if you prefer that.

E.g., to make use of all the features in [github-pr-resource](https://github.com/telia-oss/github-pr-resource)), you'll need
the following permissions for the `token-service`:
  - [statuses (`write`)](https://developer.github.com/v3/apps/permissions/#permission-on-statuses)
  - [pull requests (`write`)](https://developer.github.com/v3/apps/permissions/#permission-on-pull-requests)
  - [repository contents (`read`)](https://developer.github.com/v3/apps/permissions/#permission-on-contents)

Note that we went with two Github Apps because we did not want to generate access tokens from the `key-service` app, because
the token would have admin access to all repositories where the app was installed, and unfortunately have not found a way
to further scope down the privileges of the generated tokens. The compromise then is to have a 2nd github app (`token-service`) which has less dangerous permissions, which we can then use to generate the access tokens.

#### Secrets

This lambda uses [aws-env](https://github.com/telia-oss/aws-env) to securely populate environment variables
with their values from either AWS Secrets manager, SSM Parameter store or KMS. This makes it easy to handle
credentials in a safe manner, and we recommend using secrets manager or SSM (over KMS) to pass the Github Apps
credentials to the lambda function. Below is an example of setting up the required secrets for the example,
using Secrets Manager:

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
