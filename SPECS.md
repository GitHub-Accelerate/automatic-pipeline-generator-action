# Automatic Pipeline Generator — Specifications

## Runtime Flow
- `main.go` orchestrates argument parsing, project detection, workflow assembly, and optional commit.
- Embedded templates are accessed through Go's `embed.FS` and loaded on demand for the detected technology.
- The base generator workflow (`templates/generator.yml`) is merged with a technology-specific job before the combined definition is serialized to `.github/workflows/main.yml`.

## Project Detection
- Detection follows a well-defined order to avoid ambiguous matches: Go → Java (Maven) → Java (Gradle) → C/C++.
- Each technology ships a `detect*Project` function that inspects sentinel files (`go.mod`, `pom.xml`, `build.gradle*`, etc.).
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
- Go and C/C++ loaders reuse `modifyJobForMakefile` to pivot build/test steps to Makefile targets when present.
- Java Maven support (`java_maven.go` + `templates/java_maven.yml`):
  - Prefers `./mvnw` when the wrapper exists and ensures executable permissions are set.
  - Runs build, test, PMD, and Checkstyle goals, then archives the Maven `target` directory.
- Java Gradle support (`java_gradle.go` + `templates/java_gradle.yml`):
  - Detects Gradle projects via wrapper scripts or build descriptors.
  - Promotes `./gradlew` usage when available, adding a permission fix-up step.
  - Runs build, test, PMD, and Checkstyle tasks, then packages `build/libs` artifacts.

## Serialization and Persistence
- `generateWorkflowData` leverages a YAML encoder with indentation control to preserve formatting.
- `buildWorkflow` guarantees the `run-generator` job precedes language-specific jobs and maintains job order metadata through custom marshalers.
- `checkWorkflowChanged` avoids redundant commits by comparing the generated workflow with the existing file before writing.
- `commitAndPushWorkflow` encapsulates the go-git workflow update, using environment-provided credentials.

## Extensibility Guidelines
- Add new technology support by creating a `<language>.go` file, a template under `templates/`, and wiring detection into `main.go` following the established order.
- Keep technology loaders free of side effects besides job mutation to ensure they remain composable and testable.
- Prefer targeted helpers over inline logic for cross-language behaviors to maintain clarity and reduce duplication.
