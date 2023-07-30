include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//unassigned-enis"
}

dependency "vpc" {
  config_path = "..//vpc"
}

locals {
  common_inputs = read_terragrunt_config(find_in_parent_folders())
}

inputs = merge(local.common_inputs, {
  name_prefix = "ripper"
  subnet_ids  = dependency.vpc.outputs.private_subnets
  nr_of_enis  = 10
  private_ips = ["10.0.3.16", "10.0.3.17"]
})