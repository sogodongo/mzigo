output "vpc_id" {
  description = "ID of the created VPC."
  value       = aws_vpc.main.id
}

output "vpc_cidr" {
  description = "CIDR block of the created VPC."
  value       = aws_vpc.main.cidr_block
}

output "public_subnet_ids" {
  description = "IDs of the public subnets (one per AZ)."
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "IDs of the private subnets. EKS nodes and application pods run here."
  value       = aws_subnet.private[*].id
}

output "intra_subnet_ids" {
  description = "IDs of the intra subnets. MSK brokers and RDS run here with no internet routing."
  value       = aws_subnet.intra[*].id
}

output "nat_gateway_ids" {
  description = "IDs of the NAT gateways (one per AZ)."
  value       = aws_nat_gateway.main[*].id
}
