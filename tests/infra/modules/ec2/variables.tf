variable "vpc_id" {
  type = string
}

variable "subnet_id" {
  type = string
}

variable "name_prefix" {
  type    = string
  default = "ec2"
}

variable "tags" {
  type    = map(string)
  default = {}
}