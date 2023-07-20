include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//ecs"
}

dependency "vpc" {
  config_path = "..//vpc"
}

inputs = {
  name            = "ripper"
  vpc_id          = dependency.vpc.outputs.vpc_id
  private_subnets = dependency.vpc.outputs.private_subnets
  public_subnets  = dependency.vpc.outputs.public_subnets
}