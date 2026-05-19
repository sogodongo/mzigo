terraform {
  required_version = ">= 1.7.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.31"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Remote state backend. Teams replace this with their own S3 bucket
  # and DynamoDB lock table before first apply.
  # See: https://developer.hashicorp.com/terraform/language/settings/backends/s3
  backend "s3" {
    # bucket         = "your-tfstate-bucket"
    # key            = "mzigo/terraform.tfstate"
    # region         = "us-east-1"
    # dynamodb_table = "terraform-locks"
    # encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "mzigo"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}
