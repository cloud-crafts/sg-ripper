terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket  = "terraform-state-a4ldev"
    key     = "sg-ripper/infra/tfstate"
    region  = "us-east-1"
    profile = "A4L-DEV"
  }
}

# Configure the AWS Provider
provider "aws" {
  region  = var.aws_region
  profile = "A4L-DEV"
}