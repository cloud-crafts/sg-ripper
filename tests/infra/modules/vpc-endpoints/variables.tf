variable "vpc_id" {
  type = string
}

variable "vpc_cidr" {
  type = string
}

variable "subnets" {
  type = list(string)
}

variable "name_prefix" {
  type    = string
  default = "ec2"
}

variable "aws_region" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}