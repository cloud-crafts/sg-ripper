output "az_ids" {
  value = data.aws_availability_zones.azs.zone_ids
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}