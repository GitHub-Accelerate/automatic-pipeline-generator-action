# Automatic pipeline generator action

## How to use

Create a `GH_WORKFLOW_WRITE` fine-grained personal access tokens with access to your repos where you'll be using this action. The token requires Read and Write access to Workflows, Contents and Actions(?). 

Create a workflow with the following content:

```yaml
name: Pipeline generator

"on":
    - push
    - pull_request

permissions: write-all

    run-generator:
        outputs:
            workflow-updated: ${{ steps.run-action.outputs.workflow-updated }}
        runs-on: ubuntu-latest
        steps:
            - name: Checkout repository
              uses: actions/checkout@v5
              with:
                persist-credentials: false
            - id: run-action
              name: Run pipeline-gen action
              uses: GitHub-Accelerate/automatic-pipeline-generator-action@main
            - env:
                GH_WORKFLOW_WRITE: ${{ secrets.GH_WORKFLOW_WRITE }}
              if: steps.run-action.outputs.workflow-updated == 'true'
              name: Commit and push if workflow changed
              run: |
                git config user.name "github-actions[bot]"
                git config user.email "github-actions[bot]@users.noreply.github.com"
                git add .github/workflows/main.yml
                git commit -m "chore: update workflow"
                git push "https://x-access-token:${GH_WORKFLOW_WRITE}@github.com/${GITHUB_REPOSITORY}.git" HEAD:${GITHUB_REF#refs/heads/}

```

On a first run, the generator will detect the technology you're using a build a golden pipeline. This workflow will be updated each times there are updates to the golden pipeline. 

## Supported technologies

- Go
- Python
- Java (Maven)
- C/C++
- C#

When you want the generator to build an unsupported technology such as Rust, you can use a `Dockerfile.build` and a `Dockerfile.test` which will replace those steps in the pipeline. 

When the default build commands don't work, you can also use a `Makefile` which contains a `build` and `test` target. 

## Customization 

This action supports the following parameters:

| Parameter             | Description                                                                 | Example values                 |
|-----------------------|-----------------------------------------------------------------------------|--------------------------------|
| packages_to_install   | Packages to install. Not recommended; use a Docker image with dependencies pre-installed. | libc6-dev libgl1-mesa-dev libsdl3-dev |
| docker_image_to_use   | Docker image to use.                                                        | golang:1.25.3                  |
| item_to_build         | Override automatic detection for the item to build.                         | pom.xml, go.mod, Makefile, Dockerfile |

