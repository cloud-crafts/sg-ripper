locals {
  aws_region  = "us-east-1"
  aws_profile = "A4L-DEV"
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
provider "aws" {
  region  = "${local.aws_region}"
  profile = "${local.aws_profile}"
}
EOF
}

remote_state {
  backend = "s3"
  config  = {
    encrypt = true
    bucket  = "terraform-state-a4ldev"
    key     = "sg-ripper/${path_relative_to_include()}/terraform.tfstate"
    region  = local.aws_region
    profile = local.aws_profile
  }
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
}

inputs = {
  aws_region  = local.aws_region
  aws_profile = local.aws_profile

  tags = {
    Terraform   = "true"
    Environment = "dev"
    Project     = "sg-ripper"
  }
}