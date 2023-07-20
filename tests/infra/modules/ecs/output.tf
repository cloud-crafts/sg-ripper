output "alb_sg" {
  value = aws_security_group.alb_sg.id
}

output "alb_sg_managed" {
  value = module.alb.security_group_id
}