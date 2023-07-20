include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//ec2"
}

dependency "vpc" {
  config_path = "..//vpc"
}

inputs = {
  name      = "ripper"
  vpc_id    = dependency.vpc.outputs.vpc_id
  subnet_id = dependency.vpc.outputs.private_subnets[0]
}