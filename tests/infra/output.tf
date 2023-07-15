output "vpc_id" {
  value = module.vpc.vpc_id
}

output "public_subnet_arns" {
  value = module.vpc.public_subnets
}

output "private_subnet_arns" {
  value = module.vpc.private_subnets
}

# lambda
output "lambda_sg" {
  value = aws_security_group.lambda_sg.id
}

# ec2
output "ec2_sg" {
  value = aws_security_group.ec2_sg.id
}

# ecs
output "alb_sg" {
  value = aws_security_group.alb_sg.id
}

output "alb_sg_managed" {
  value = module.alb.security_group_id
}