# Golang Project Template

## Project Structure

This is a simple layout for Golang projects based on the [community standard][layout],
with added machinery for running in Docker.

## Setup Instructions

1. Select "use this template" to create a new Github repo based on this one.
2. Replace `golang-template` with the name of the project.
3. Optional customization:
    1. Set the package name in `go.mod` if deploying under a different Github user/organization.
    2. Set the image name in `docker-compose.yml` if deploying to a different Docker registry or repository.
    3. If using dev containers, copy `.devcontainer.example` to `.devcontainer`.
4. Ensure the software license is correct.
5. Commit changes and begin developing.

## Docker Compose Workflow

Example run command:

```bash
docker-compose run --build --rm app
```

Available services:

* `app` - contains the compiled binaries, ready for production deployment
* `dev` - dev container for running arbitrary build and debug commands
* `test` - run tests in dev container
* `cover` - run tests with code coverage


[layout]: https://github.com/golang-standards/project-layout
