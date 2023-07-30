output "available_address" {
  value = data.aws_subnet.private_subnet.available_ip_address_count
}

output "cidr" {
  value = data.aws_subnet.private_subnet.cidr_block
}