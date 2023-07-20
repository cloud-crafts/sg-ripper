include {
  path = find_in_parent_folders()
}

terraform {
  source = "../../modules//vpc"
}

dependency "common" {
  config_path = "..//common"
}

inputs = {
  name_prefix = "my-vpc"
  cidr        = "10.0.0.0/16"

  az_ids          = slice(dependency.common.outputs.az_ids, 0, 3)
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  tags = {
    Terraform   = "true"
    Environment = "dev"
  }
}