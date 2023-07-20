output "lambda_sg" {
  value = aws_security_group.lambda_sg.id
}

output "another_lambda_sg" {
  value = aws_security_group.another_lambda_sg.id
}

output "common_lambda_sg" {
  value = aws_security_group.common_sg.id
}