include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//rds-aurora"
}

dependency "vpc" {
  config_path = "..//vpc"
}

inputs = {
  name_prefix     = "ripper"
  vpc_id          = dependency.vpc.outputs.vpc_id
  private_subnets = dependency.vpc.outputs.private_subnets
  vpc_cidr        = dependency.vpc.outputs.vpc_cidr
}