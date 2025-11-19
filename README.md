# Automatic pipeline generator action

## How to use

Create a `GH_WORKFLOW_WRITE` fine-grained personal access tokens with access to your repos where you'll be using this action. The token requires Read and Write access to Workflows, Contents and Actions(?). 

Create a workflow with the following content:

```yaml
name: Pipeline generator
on:
  - push
  - pull_request
permissions: write-all
jobs:
  run-generator:
    runs-on: ubuntu-latest
    outputs:
      workflow-updated: ${{ steps.run-action.outputs.workflow-updated }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v5
        with:
          persist-credentials: false
      - name: Run pipeline-gen action
        id: run-action
        uses: GitHub-Accelerate/automatic-pipeline-generator-action@main
        env:
          GH_WORKFLOW_WRITE: ${{ secrets.GH_WORKFLOW_WRITE }}
```

On a first run, the generator will detect the technology you're using a build a golden pipeline. This workflow will be updated each times there are updates to the golden pipeline. 

## Supported technologies

- Go
- Python
- Java (Maven)
- C/C++
- C#
- PHP
- Docker

When you want the generator to build an unsupported technology such as Rust, you can use a `Dockerfile.build` and a `Dockerfile.test` which will replace those steps in the pipeline. 

When the default build commands don't work, you can also use a `Makefile` which contains a `build` and `test` target. 

Note: Filenames are case-sensitive. 

## Customization 

This action supports the following input parameters:

| Parameter             | Description                                                                 | Example values                 |
|-----------------------|-----------------------------------------------------------------------------|--------------------------------|
| packages_to_install   | Packages to install. Not recommended; use a Docker image with dependencies pre-installed. | libc6-dev libgl1-mesa-dev libsdl3-dev |
| docker_image_to_use   | Docker image to use.                                                        | golang:1.25.3                  |
| language_version      | Sets the language version for the setup action                              | 1.25.3                         |
| item_to_build         | Override automatic detection for the item to build.                         | pom.xml, go.mod, Makefile, Dockerfile |

