# reference
# https://stackoverflow.com/questions/68658353/

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.91.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = ">= 3.0.2"
    }
  }
}

# provider configured with environment vars
provider "aws" {}

variable "AWS_REGION" {}

resource "aws_ecr_repository" "repo" {
  name = "pastebin"
}

# get authorization credentials to push to ecr
data "aws_ecr_authorization_token" "token" {}

provider "docker" {
  registry_auth {
    address  = data.aws_ecr_authorization_token.token.proxy_endpoint
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

# build docker image
resource "docker_image" "pastebin" {
  name = "${aws_ecr_repository.repo.repository_url}:latest"
  build {
    context = "./server"
  }
}

# push image to ecr repo
resource "docker_registry_image" "upload" {
  name = docker_image.pastebin.name
}
