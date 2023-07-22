# sg-ripper - Integration Tests

Integration tests for `sg-ripper` rely on the existence of an AWS account.

*Warning: Running integration tests require the deployment of one or more stacks from `infra`. Provisioning infrastructure will generate AWS costs!*

## Running Integration Tests

1. Deploy `infra` (see: [README](infra/README.md))
2. Execute integration all integration tests (for a subset, see: [README](integration/README.md):  

```shell
go test ./integration/... -v
```

