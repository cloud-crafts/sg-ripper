locals {
  db_port = 3306
}

module "cluster" {
  source = "terraform-aws-modules/rds-aurora/aws"

  name           = "${var.name_prefix}-rds-aurora-mysql"
  engine         = "aurora-mysql"
  engine_version = "8.0"
  instance_class = "db.t4g.medium"

  vpc_id               = var.vpc_id
  security_group_rules = {
    ex1_ingress = {
      cidr_blocks = [var.vpc_cidr]
    }
  }

  subnets                = var.private_subnets
  create_db_subnet_group = true

  storage_encrypted = false
  apply_immediately = true

  master_username = "ripper"

  instances = {
    one = {}
  }

  skip_final_snapshot = true

  tags = var.tags
}