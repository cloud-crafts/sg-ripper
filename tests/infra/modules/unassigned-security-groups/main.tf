resource "aws_security_group" "unassigned" {
  count       = var.nr_of_security_groups
  name        = "${var.name_prefix}-unassigned-${count.index}"
  description = "Unassigned Security Group."
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}