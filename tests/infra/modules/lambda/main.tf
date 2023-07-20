module "lambda_function" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = "${var.name_prefix}-lambda"
  description   = "Test Lambda function in VPC for sg-ripper"
  handler       = "main.lambda_handler"
  runtime       = "python3.9"

  vpc_subnet_ids         = var.private_subnets
  vpc_security_group_ids = [aws_security_group.lambda_sg.id, aws_security_group.common_sg.id]
  attach_network_policy  = true

  source_path = var.source_path

  tags = var.tags
}

module "another_lambda_function" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = "${var.name_prefix}-lambda-2"
  description   = "Test Lambda function in VPC for sg-ripper"
  handler       = "main.lambda_handler"
  runtime       = "python3.9"

  vpc_subnet_ids         = var.private_subnets
  vpc_security_group_ids = [aws_security_group.another_lambda_sg.id, aws_security_group.common_sg.id]
  attach_network_policy  = true

  source_path = var.source_path

  tags = var.tags
}

resource "aws_security_group" "lambda_sg" {
  name        = "${var.name_prefix}-lambda-sg"
  description = "Security Group attached to the sg-ripper-test-lambda function."
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

resource "aws_security_group" "another_lambda_sg" {
  name        = "${var.name_prefix}-another-lambda-sg"
  description = "Security Group attached to the sg-ripper-test-lambda-2 function."
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

resource "aws_security_group" "common_sg" {
  name        = "${var.name_prefix}-lambda-common-sg"
  description = "Security Group attached to all sg-ripper-test-lambda functions."
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}