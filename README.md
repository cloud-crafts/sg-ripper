# sg-ripper

`sg-ripper` is a tool that can be used to detect which Security Group is in use in an AWS account and which one can 
be removed. 

`sg-ripper` gives detailed information about which ENI (Elastic Network Interfaces) is attached to which 
Security Group and what kind of AWS resource is using that ENI. It also can detect which ENIs are potentially stuck
after the removal of the resource.

## Usage

```shell
Usage:
  sg-ripper [command]

Available Commands:
  help        Help about any command
  list        List Security Groups with Details
  list-eni    List Elastic Network Interfaces with Details
  remove      Remove unused security groups.

Flags:
  -h, --help             help for sg-ripper
      --profile string   [Optional] Profile.
      --region string    [Optional] AWS Region.
  -v, --version          version for sg-ripper

Use "sg-ripper [command] --help" for more information about a command.

```

Examples:

```shell
sg-ripper list --sg sg-12354
```

```shell
sg-ripper list-eni --eni eni-1234
```

## Building

- Windows:  

```shell
go build -o dist/sg-ripper.exe
```

- Linux/MacOS:  

```shell
go build -o dist/sg-ripper
```