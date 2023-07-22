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
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List Security Groups with Details

Flags:
  -h, --help             help for sg-ripper
      --profile string   [Optional] Profile.
      --region string    [Optional] AWS Region.
  -v, --version          version for sg-ripper
```

Example:

```shell
sg-ripper --sg sg-12354
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