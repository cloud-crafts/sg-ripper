module "ec2_instance" {
  source = "terraform-aws-modules/ec2-instance/aws"

  name = "${var.name_prefix}-host"

  instance_type          = "t2.micro"
  vpc_security_group_ids = [aws_security_group.ec2_sg.id]
  subnet_id              = var.subnet_id

  tags = var.tags
}

resource "aws_security_group" "ec2_sg" {
  name        = "${var.name_prefix}-sg"
  description = "Security Group attached to the sg-ripper-test-ec2 EC2 instance."
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}