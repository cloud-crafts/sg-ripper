output "unassigned_eni_id" {
  value = aws_network_interface.eni[*].id
}

output "unassigned_eni_ips" {
  value = {
    for eni in aws_network_interface.eni : eni.id => eni.private_ip
  }
}