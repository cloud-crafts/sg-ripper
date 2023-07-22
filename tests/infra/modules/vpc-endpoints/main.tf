resource "aws_vpc_endpoint" "ec2" {
  vpc_id            = var.vpc_id
  subnet_ids        = var.subnets
  service_name      = "com.amazonaws.${var.aws_region}.ec2"
  vpc_endpoint_type = "Interface"

  security_group_ids = [
    aws_security_group.ec2_vpce_sg.id,
  ]

  private_dns_enabled = true
}

resource "aws_security_group" "ec2_vpce_sg" {
  name        = "${var.name_prefix}-vpce-sg"
  description = "Security Group attached to the VPCE for EC2."
  vpc_id      = var.vpc_id

  ingress {
    from_port = 443
    to_port   = 443
    protocol  = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}