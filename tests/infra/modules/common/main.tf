# Retrieve availability zones for the current region
data "aws_availability_zones" "azs" {
  state = "available"
}

# Retrieve current account ID
data "aws_caller_identity" "current" {}