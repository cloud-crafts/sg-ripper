# Infra for Integration Tests

This Terraform project should be used for setting up the infrastructure for the integration tests. 
It is recommended to do the setup in an empty account.

## Deployment

```shell
cd live
terragrunt run-all apply
```