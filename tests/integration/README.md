# sg-ripper - Integration Tests

## Run a Subset of Integration Tests

In order to run a subset of integration tests, first we have to provision the infrastructure for it.

For example, let's run `ecs` integration tests:

1. Provision the infrastructure by going into `tests/infra/live/ecs` folder and running `terragrunt apply-all`
2. Execute the test suite: `go test -v sg-ripper/tests/integration/ecs`
3. Tear down the infrastructure by going into `tests/infra/live/ecs` folder and running `terragrunt destroy-all`