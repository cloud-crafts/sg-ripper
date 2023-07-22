output "alb_sg" {
  value = aws_security_group.alb_sg.id
}

output "alb_sg_managed" {
  value = module.alb.security_group_id
}

output "container_sg_id" {
  value = module.ecs.services[local.container_name]["security_group_id"]
}