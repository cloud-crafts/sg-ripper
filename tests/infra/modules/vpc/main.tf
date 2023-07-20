module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = "${var.name_prefix}-vpc"
  cidr = var.cidr

  azs             = var.az_ids
  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets

  create_igw         = true
  single_nat_gateway = true
  enable_nat_gateway = true

  tags = var.tags
}