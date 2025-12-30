# Automatic Pipeline Generator — Specifications

## Runtime Flow
- `main.go` orchestrates argument parsing, project detection, workflow assembly, and optional commit.
- Embedded templates are accessed through Go's `embed.FS` and loaded on demand for the detected technology.
- The base generator workflow (`templates/generator.yml`) is merged with a technology-specific job before the combined definition is serialized to `.github/workflows/main.yml`.

## Project Detection
- Detection follows a well-defined order to avoid ambiguous matches: Python → Go → Java (Maven) → Java (Gradle) → C# → C/C++.
- Each technology ships a `detect*Project` function that inspects sentinel files (`requirements.txt`, `pyproject.toml`, `go.mod`, `pom.xml`, `build.gradle*`, `.sln`, etc.).
- Detection helpers return a boolean and a human-readable indicator to aid logging and traceability.

## Template Loading Pattern
- A dedicated `load*JobTemplate` function exists per technology file. The function:
  - Reads the matching template from `templates/`.
  - Unmarshals YAML into `GitHubWorkflow` and extracts the single job entry.
  - Applies shared customizations (`applyFetchDepth`, `applyPackagesToInstall`).
  - Performs technology-specific adjustments (Makefile overrides, wrapper promotion, etc.).
This pattern keeps parsing, transformation, and specialization isolated per technology while reusing shared helpers.

## Shared Customizations
- `applyFetchDepth` injects `fetch-depth` on the checkout step when requested.
- `applyPackagesToInstall` inserts a package installation step directly after the checkout step.
- `insertStepAfterCheckout` (in `java_common.go`) centralizes post-checkout step insertion to maintain execution order.
- `replaceCommandPrefix` rewrites run command prefixes while preserving indentation and arguments, ensuring wrapper-aware replacements stay idempotent.

## Language-Specific Behavior
- Python support (`python.go` + `templates/python.yml`):
  - Detects Python projects via `requirements.txt`, `pyproject.toml`, `setup.py`, or `Pipfile`.
  - Auto-detects Poetry (via `[tool.poetry]` in `pyproject.toml`) and switches to Poetry commands.
  - Auto-detects uv (via `uv.lock`) and uses uv for faster package installation.
  - Defaults to pip for standard Python projects.
  - Always includes ruff for linting and code quality checks.
  - Builds distributable packages using `python -m build` and uploads to artifacts.
  - PyPI publishing can be added by including secrets (PYPI_TOKEN) and using the workflow's build artifacts.
- Go and C/C++ loaders reuse `modifyJobForMakefile` to pivot build/test steps to Makefile targets when present.
- Java Maven support (`java_maven.go` + `templates/java_maven.yml`):
  - Prefers `./mvnw` when the wrapper exists and ensures executable permissions are set.
  - Runs build, test, PMD, and Checkstyle goals, then archives the Maven `target` directory.
- Java Gradle support (`java_gradle.go` + `templates/java_gradle.yml`):
  - Detects Gradle projects via wrapper scripts or build descriptors.
  - Promotes `./gradlew` usage when available, adding a permission fix-up step.
  - Runs build, test, PMD, and Checkstyle tasks, then packages `build/libs` artifacts.
- C# support (`csharp.go` + `templates/csharp.yml`):
  - Detects C# projects via `.sln` files at the repository root.
  - Parses solution files to extract `.csproj` project paths.
  - Uses `dotnet restore`, `dotnet build`, `dotnet test`, and `dotnet publish` commands.
  - Publishes built artifacts from the `./publish` directory.

## Serialization and Persistence
- `generateWorkflowData` leverages a YAML encoder with indentation control to preserve formatting.
- `buildWorkflow` guarantees the `run-generator` job precedes language-specific jobs and maintains job order metadata through custom marshalers.
- `checkWorkflowChanged` avoids redundant commits by comparing the generated workflow with the existing file before writing.
- `commitAndPushWorkflow` encapsulates the go-git workflow update, using environment-provided credentials.

## Extensibility Guidelines
- Add new technology support by creating a `<language>.go` file, a template under `templates/`, and wiring detection into `main.go` following the established order.
- Keep technology loaders free of side effects besides job mutation to ensure they remain composable and testable.
- Prefer targeted helpers over inline logic for cross-language behaviors to maintain clarity and reduce duplication.
