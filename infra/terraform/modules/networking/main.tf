locals {
  az_count = length(var.availability_zones)

  # Subnet CIDR calculation.
  # /16 VPC split into /20 subnets gives 16 subnets of 4094 usable IPs each.
  # Private subnets: workloads (EKS nodes, RDS, MSK)
  # Public subnets:  NAT gateway EIPs, load balancers
  # Intra subnets:   MSK brokers (no internet access, no NAT)
  private_subnet_cidrs = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 4, i)]
  public_subnet_cidrs  = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 4, i + local.az_count)]
  intra_subnet_cidrs   = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 4, i + local.az_count * 2)]
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.name_prefix}-vpc"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.name_prefix}-igw"
  }
}

# Public subnets: one per AZ, route to internet gateway directly.
resource "aws_subnet" "public" {
  count = local.az_count

  vpc_id                  = aws_vpc.main.id
  cidr_block              = local.public_subnet_cidrs[count.index]
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name                     = "${var.name_prefix}-public-${var.availability_zones[count.index]}"
    "kubernetes.io/role/elb" = "1"
  }
}

# Elastic IPs for NAT gateways.
resource "aws_eip" "nat" {
  count  = local.az_count
  domain = "vpc"

  tags = {
    Name = "${var.name_prefix}-nat-eip-${var.availability_zones[count.index]}"
  }

  depends_on = [aws_internet_gateway.main]
}

# One NAT gateway per AZ.
# Single NAT gateway is cheaper but creates a cross-AZ dependency:
# if the NAT gateway's AZ goes down, all private subnets lose egress.
# Per-AZ NAT gateways eliminate that dependency for production workloads.
resource "aws_nat_gateway" "main" {
  count = local.az_count

  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id

  tags = {
    Name = "${var.name_prefix}-nat-${var.availability_zones[count.index]}"
  }

  depends_on = [aws_internet_gateway.main]
}

# Private subnets: EKS nodes, application pods.
resource "aws_subnet" "private" {
  count = local.az_count

  vpc_id            = aws_vpc.main.id
  cidr_block        = local.private_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name                              = "${var.name_prefix}-private-${var.availability_zones[count.index]}"
    "kubernetes.io/role/internal-elb" = "1"
  }
}

# Intra subnets: MSK brokers and RDS. No NAT, no internet routing.
# MSK brokers do not need internet access. Removing NAT reduces the
# blast radius of a broker compromise.
resource "aws_subnet" "intra" {
  count = local.az_count

  vpc_id            = aws_vpc.main.id
  cidr_block        = local.intra_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name = "${var.name_prefix}-intra-${var.availability_zones[count.index]}"
  }
}

# Public route table: default route to internet gateway.
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${var.name_prefix}-public-rt"
  }
}

resource "aws_route_table_association" "public" {
  count          = local.az_count
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Private route tables: one per AZ, default route through AZ-local NAT gateway.
resource "aws_route_table" "private" {
  count  = local.az_count
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main[count.index].id
  }

  tags = {
    Name = "${var.name_prefix}-private-rt-${var.availability_zones[count.index]}"
  }
}

resource "aws_route_table_association" "private" {
  count          = local.az_count
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

# Intra route table: no default route. Intra subnets are fully isolated.
resource "aws_route_table" "intra" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.name_prefix}-intra-rt"
  }
}

resource "aws_route_table_association" "intra" {
  count          = local.az_count
  subnet_id      = aws_subnet.intra[count.index].id
  route_table_id = aws_route_table.intra.id
}

# VPC Flow Logs: record all traffic for security auditing.
# Flow logs to CloudWatch are used for incident investigation and
# anomaly detection. The IAM role is created inline here because
# it is tightly coupled to this specific VPC resource.
resource "aws_flow_log" "main" {
  vpc_id          = aws_vpc.main.id
  traffic_type    = "ALL"
  iam_role_arn    = aws_iam_role.flow_log.arn
  log_destination = aws_cloudwatch_log_group.flow_log.arn

  tags = {
    Name = "${var.name_prefix}-flow-logs"
  }
}

resource "aws_cloudwatch_log_group" "flow_log" {
  name              = "/mzigo/${var.name_prefix}/vpc-flow-logs"
  retention_in_days = 30
}

resource "aws_iam_role" "flow_log" {
  name = "${var.name_prefix}-flow-log-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "vpc-flow-logs.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "flow_log" {
  name = "${var.name_prefix}-flow-log-policy"
  role = aws_iam_role.flow_log.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
      ]
      Resource = "*"
    }]
  })
}
