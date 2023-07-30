data "aws_subnet" "private_subnet" {
  id = element(var.subnet_ids, 0)
}

resource "aws_network_interface" "eni" {
  count           = var.nr_of_enis
  subnet_id       = element(var.subnet_ids, count.index)
}

resource "aws_network_interface" "eni-with-ips" {
  subnet_id   = element(var.subnet_ids, 0)
  private_ips = ["10.0.1.128", "10.0.1.129", "10.0.1.130"]
}