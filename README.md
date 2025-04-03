# Pastebin

Pastebin is a simple web application written in Golang and designed for use with
a MongoDB backend.

![screenshot](./pastebin.png)

## Project Layout

This repo contains everything necessary to deploy Pastebin locally or to build
the Pastebin container image and push it to AWS ECR.

`db/` contains the Dockerfile and MongoDB `init-mongo.js` necessary for
a local test instance of MongoDB. The Mongo image is NOT built and pushed to AWS
by default. It is expected one would use a MongoDB server running on an EC2, or
perhaps a DocumentDB instance.

`server/` contains the static HTML, CSS, JS, and Golang source for the Pastebin
web application, as well as a Dockerfile for building the application container.

`push-to-ecr.tf` is a Terraform configuration file for building and pushing the
Pastebin container to AWS. It expects your environment to contain three ENV
variables for authentication to AWS: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
and `AWS_REGION`.

`docker-compose.yaml` contains the configuration necessary to build and deploy
the pastebin and throwaway Mongo containers for local testing. To ensure that
the containers are actually rebuilt and not cached, I recommend the following
commands:

```sh
# bring local instance up
docker-compose up -d

# bring local instance down
docker-compose down --rmi all -v --remove-orphans
```

Note that the local Mongo instance's data is not persisted in the default
configuration.

## Deployment and/or Customization

As you will see in the `docker-compose` file, the Pastebin application expects
two environment variables:

- `DB_CONN_STRING`
- `DISABLE_HTML_ESCAPE`

`DB_CONN_STRING` tells the application how to connect to its backend. It will
likely contain credentials, and should be treated as a secret.

`DISABLE_HTML_ESCAPE` toggles whether the application performs input sanitation
on user-submitted data. Setting it equal to 1 will cause the application to be
vulnerable to stored XSS.