include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//vpc-endpoints"
}

dependency "vpc" {
  config_path = "..//vpc"
}

locals {
  common_inputs = read_terragrunt_config(find_in_parent_folders())
}

inputs = merge(local.common_inputs, {
  name_prefix = "ripper"
  vpc_id      = dependency.vpc.outputs.vpc_id
  subnets     = dependency.vpc.outputs.private_subnets
})