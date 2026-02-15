# Contributor Requirements

This document outlines the requirements and workflow for contributing to the Kubeflow Model Registry project.

## Overview

The Model Registry follows Kubeflow's standard contribution process, including DCO sign-off, OWNERS-based review, and automated CI/CD checks.

## Developer Certificate of Origin (DCO)

All contributions must be signed off to certify that you have the right to submit the code under the project's license.

### Sign-off Format

```
Signed-off-by: Your Name <your.email@example.com>
```

### How to Sign Off

```bash
# Add sign-off to commit
git commit -s -m "Your commit message"

# Amend existing commit with sign-off
git commit --amend -s

# Configure git to auto-sign
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### Git Hook for Auto Sign-off

Create `.git/hooks/prepare-commit-msg`:

```bash
#!/bin/sh

NAME=$(git config user.name)
EMAIL=$(git config user.email)

if [ -z "$NAME" ]; then
    echo "empty git config user.name"
    exit 1
fi

if [ -z "$EMAIL" ]; then
    echo "empty git config user.email"
    exit 1
fi

git interpret-trailers --if-exists doNothing --trailer \
    "Signed-off-by: $NAME <$EMAIL>" \
    --in-place "$1"
```

Make it executable:
```bash
chmod +x .git/hooks/prepare-commit-msg
```

## OWNERS Files

The project uses Kubernetes-style OWNERS files to manage code review and approval.

### Structure

```yaml
# OWNERS
approvers:
  - username1
  - username2
reviewers:
  - username1
  - username2
  - username3
```

### Current Approvers

| Username | Area |
|----------|------|
| Al-Pragliola | General, UI |
| andreyvelich | General |
| ckadner | General |
| ederign | General, UI |
| pboyd | General |
| rareddy | General |
| tarilabs | General |
| Tomcli | General |
| zijianjoy | General |

### OWNERS Files in Repository

```
model-registry/
├── OWNERS                      # Root approvers/reviewers
├── clients/ui/OWNERS          # UI-specific owners
└── manifests/kustomize/OWNERS # Manifest owners
```

## Pull Request Workflow

### 1. Create an Issue

Before starting work, open an issue to discuss:
- Bug fixes
- New features
- Significant changes

Use the appropriate [issue template](https://github.com/kubeflow/model-registry/issues/new/choose).

### 2. Fork and Clone

```bash
# Fork via GitHub UI, then:
git clone https://github.com/YOUR_USERNAME/model-registry
cd model-registry
git remote add upstream https://github.com/kubeflow/model-registry
```

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 4. Make Changes

- Follow the [style guide](./style-guide.md)
- Add tests for new functionality
- Update documentation as needed

### 5. Commit with Sign-off

```bash
git add .
git commit -s -m "Description of changes"
```

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Create a PR via GitHub, referencing the related issue.

### 7. Address Review Feedback

- Respond to reviewer comments
- Push additional commits as needed
- Ensure all CI checks pass

### 8. Merge

Once approved by OWNERS and CI passes, the PR will be merged.

## CI/CD Checks

All PRs must pass the following checks:

### Required Checks

| Check | Description |
|-------|-------------|
| DCO | Developer Certificate of Origin sign-off |
| build | Go build success |
| test | Go unit tests |
| lint | Go linting (golangci-lint) |
| openapi-validate | OpenAPI specification validation |
| frontend-lint | ESLint + Prettier for TypeScript |
| frontend-test | Jest and Cypress tests |
| bff-lint | BFF Go linting |
| bff-test | BFF unit tests |

### Running Checks Locally

```bash
# Go checks
make vet
make lint
make test

# Frontend checks
cd clients/ui/frontend
npm run lint
npm run test

# BFF checks
cd clients/ui/bff
make lint
make test
```

## Commit Message Guidelines

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Description |
|------|-------------|
| feat | New feature |
| fix | Bug fix |
| docs | Documentation changes |
| style | Code style changes (formatting) |
| refactor | Code refactoring |
| test | Adding/updating tests |
| chore | Maintenance tasks |

### Examples

```
feat(api): add model version filtering endpoint

Adds a new endpoint for filtering model versions by custom properties.

Closes #123

Signed-off-by: Developer Name <dev@example.com>
```

```
fix(frontend): resolve pagination issue in model list

The pagination component was not updating correctly when
switching between pages. This fix ensures the page state
is properly synchronized.

Fixes #456

Signed-off-by: Developer Name <dev@example.com>
```

## Finding Issues to Work On

### Good First Issues

Look for issues labeled:
- `good first issue`
- `help wanted`

Browse: [Good First Issues](https://github.com/kubeflow/model-registry/labels/good%20first%20issue)

### Issue Workflow

1. Comment on the issue to express interest
2. Wait for assignment or approval
3. Create PR referencing the issue
4. Use `Fixes #123` or `Closes #123` in commit/PR

## Community

### Meetings

The Kubeflow Model Registry has bi-weekly community meetings. Check the [Kubeflow Community Calendar](https://www.kubeflow.org/docs/about/community/#kubeflow-community-calendar).

### Communication

- **Slack**: [Kubeflow Slack](https://kubeflow.slack.com) - `#model-registry` channel
- **Mailing List**: kubeflow-discuss@googlegroups.com

### Code of Conduct

All contributors must follow the [Kubeflow Code of Conduct](https://www.kubeflow.org/docs/about/contributing/#follow-the-code-of-conduct).

## License

Contributions are licensed under the Apache License 2.0. By contributing, you agree that your contributions will be licensed under this license.

---

[Back to Guides Index](./README.md) | [Previous: Developer Guide](./developer-guide.md) | [Next: Style Guide](./style-guide.md)
