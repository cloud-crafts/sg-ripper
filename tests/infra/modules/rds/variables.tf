variable "name_prefix" {
  type    = string
  default = "vpc"
}

variable "vpc_cidr" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "private_subnets" {
  type = set(string)
}

variable "tags" {
  type    = map(string)
  default = {}
}