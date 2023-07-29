variable "name_prefix" {
  type    = string
  default = "vpc"
}

variable "vpc_id" {
  type = string
}

variable "nr_of_security_groups" {
  type = number
}

variable "tags" {
  type    = map(string)
  default = {}
}