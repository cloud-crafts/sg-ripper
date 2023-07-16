module "lambda_function" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = "sg-ripper-test-lambda"
  description   = "Test Lambda function in VPC for sg-ripper"
  handler       = "main.lambda_handler"
  runtime       = "python3.9"

  vpc_subnet_ids         = module.vpc.private_subnets
  vpc_security_group_ids = [aws_security_group.lambda_sg.id, aws_security_group.common_sg.id]
  attach_network_policy  = true

  source_path = "lambda"

  tags = {
    Name = "sg-ripper-test-lambda"
  }
}

module "another_lambda_function" {
  source = "terraform-aws-modules/lambda/aws"

  function_name = "sg-ripper-test-lambda-2"
  description   = "Test Lambda function in VPC for sg-ripper"
  handler       = "main.lambda_handler"
  runtime       = "python3.9"

  vpc_subnet_ids         = module.vpc.private_subnets
  vpc_security_group_ids = [aws_security_group.another_lambda_sg.id, aws_security_group.common_sg.id]
  attach_network_policy  = true

  source_path = "lambda"

  tags = {
    Name = "sg-ripper-test-lambda"
  }
}

resource "aws_security_group" "lambda_sg" {
  name        = "lambda-sg"
  description = "Security Group attached to the sg-ripper-test-lambda function."
  vpc_id      = module.vpc.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "sg-ripper-lambda-sg"
  }
}

resource "aws_security_group" "another_lambda_sg" {
  name        = "another-lambda-sg"
  description = "Security Group attached to the sg-ripper-test-lambda-2 function."
  vpc_id      = module.vpc.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "sg-ripper-lambda-sg-2"
  }
}

resource "aws_security_group" "common_sg" {
  name        = "lambda-common-sg"
  description = "Security Group attached to all sg-ripper-test-lambda functions."
  vpc_id      = module.vpc.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "common-sg-ripper-lambda"
  }
}