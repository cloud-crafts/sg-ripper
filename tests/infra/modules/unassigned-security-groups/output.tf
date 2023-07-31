output "unassigned_sg_id" {
  value = {
    for sg in aws_security_group.unassigned : sg.name => sg.id
  }
}