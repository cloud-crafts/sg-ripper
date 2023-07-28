locals {
  db_port = 3306
}

module "db" {
  source = "terraform-aws-modules/rds/aws"

  identifier = "${var.name_prefix}-rds-mysql"

  engine            = "mysql"
  engine_version    = "8.0"
  instance_class    = "db.t2.micro"
  allocated_storage = 5

  db_name  = "demodb"
  username = "ripper"
  port     = "3306"

  iam_database_authentication_enabled = true

  vpc_security_group_ids = [aws_security_group.rds_sg.id]

  # DB subnet group
  create_db_subnet_group = true
  subnet_ids             = var.private_subnets

  # DB parameter group
  family = "mysql8.0"

  # DB option group
  major_engine_version = "8.0"

  storage_encrypted = false

  tags = var.tags
}

resource "aws_security_group" "rds_sg" {
  name        = "${var.name_prefix}-rds-mysql-sg"
  description = "Security Group attached to the MYSQL DB."
  vpc_id      = var.vpc_id

  ingress {
    from_port   = local.db_port
    to_port     = local.db_port
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}