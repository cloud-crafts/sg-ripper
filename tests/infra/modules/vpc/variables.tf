variable "az_ids" {
  type = set(string)
}

variable "name_prefix" {
  type    = string
  default = "vpc"
}

variable "cidr" {
  type = string
}

variable "private_subnets" {
  type = set(string)
}

variable "public_subnets" {
  type = set(string)
}

variable "tags" {
  type    = map(string)
  default = {}
}