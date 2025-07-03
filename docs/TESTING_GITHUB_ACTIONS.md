# Testing GitHub Actions with Act

This document explains how to use `act` to test GitHub Actions workflows locally before pushing to GitHub.

## What is Act?

Act is a tool that allows you to run your GitHub Actions locally using Docker. It reads your workflow files and simulates the GitHub Actions environment on your local machine.

## Installation

### macOS
```bash
brew install act
```

### Linux
```bash
# Using the install script
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Or using package managers
# Ubuntu/Debian
sudo apt install act

# Arch Linux
yay -S act
```

### Windows
```bash
# Using Chocolatey
choco install act-cli

# Using Scoop
scoop install act

# Using winget
winget install nektos.act
```

## Prerequisites

- Docker must be installed and running
- Git repository with `.github/workflows/` directory

## Basic Usage

### Run All Workflows
```bash
# Run all workflows for push event (default)
act

# Run all workflows for pull request event
act pull_request

# Run all workflows for workflow_dispatch event
act workflow_dispatch
```

### Run Specific Workflows

For our kubectl-nuke-go project:

```bash
# Test PR workflow
act pull_request -W .github/workflows/pr.yml

# Test Go CI workflow
act push -W .github/workflows/go.yml

# Test release workflow
act workflow_dispatch -W .github/workflows/release.yml
```

### Dry Run (List Jobs Without Running)
```bash
# See what would run without actually running
act -n

# Dry run for specific workflow
act pull_request -W .github/workflows/pr.yml -n
```

## Advanced Usage

### Using Custom Event Files

Create event files to simulate specific scenarios:

#### Pull Request Event
```bash
# Create PR event file
cat > .github/workflows/pr-event.json << EOF
{
  "pull_request": {
    "number": 123,
    "head": {
      "ref": "feature/new-feature",
      "sha": "abc123def456"
    },
    "base": {
      "ref": "main",
      "sha": "def456abc123"
    },
    "title": "Add new feature",
    "body": "This PR adds a new feature"
  }
}
EOF

# Use the event file
act pull_request -e .github/workflows/pr-event.json
```

#### Push Event
```bash
# Create push event file
cat > .github/workflows/push-event.json << EOF
{
  "ref": "refs/heads/main",
  "repository": {
    "name": "kubectl-nuke-go",
    "full_name": "codesenju/kubectl-nuke-go"
  },
  "commits": [
    {
      "id": "abc123",
      "message": "feat: add new functionality"
    }
  ]
}
EOF

# Use the event file
act push -e .github/workflows/push-event.json
```

### Environment Variables and Secrets

#### Set Environment Variables
```bash
# Set environment variables
act -e GOOS=linux -e GOARCH=amd64

# Use environment file
echo "GOOS=linux" > .env
echo "GOARCH=amd64" >> .env
act --env-file .env
```

#### Set Secrets
```bash
# Set secrets via command line
act -s GITHUB_TOKEN=your_token_here

# Use secrets file
echo "GITHUB_TOKEN=your_token_here" > .secrets
act --secret-file .secrets
```

### Platform Selection

Act supports different runner images:

```bash
# Use specific platform
act -P ubuntu-latest=catthehacker/ubuntu:act-latest

# Use different platform for different jobs
act -P ubuntu-latest=catthehacker/ubuntu:act-latest -P windows-latest=catthehacker/ubuntu:act-latest
```

## Project-Specific Examples

### Testing Our Workflows

#### 1. Test PR Workflow
```bash
# Basic PR test
act pull_request -W .github/workflows/pr.yml

# Test with verbose output
act pull_request -W .github/workflows/pr.yml -v

# Test with specific Go version (if needed)
act pull_request -W .github/workflows/pr.yml -e GO_VERSION=1.22
```

#### 2. Test Go CI Workflow
```bash
# Test push to main
act push -W .github/workflows/go.yml

# Test pull request trigger
act pull_request -W .github/workflows/go.yml

# Test with custom event
act push -W .github/workflows/go.yml -e .github/workflows/push-event.json
```

#### 3. Test Release Workflow
```bash
# Test workflow_dispatch
act workflow_dispatch -W .github/workflows/release.yml

# Test with inputs (if your workflow has inputs)
cat > release-input.json << EOF
{
  "inputs": {
    "version": "v1.0.0",
    "prerelease": false
  }
}
EOF
act workflow_dispatch -W .github/workflows/release.yml -e release-input.json
```

### Common Testing Scenarios

#### Test Make Commands
Since our workflows use `make test-all`, you can verify this works:

```bash
# Test the PR workflow which includes make test-all
act pull_request -W .github/workflows/pr.yml

# If you want to test just the make command locally first
make test-all
```

#### Test Cross-Platform Builds
```bash
# Our workflows build for multiple platforms
# Act will test this in the Linux container
act push -W .github/workflows/go.yml -v
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Docker Permission Issues
```bash
# Add your user to docker group (Linux)
sudo usermod -aG docker $USER
# Then logout and login again
```

#### 2. Large Docker Images
```bash
# Use smaller images for faster testing
act -P ubuntu-latest=catthehacker/ubuntu:act-20.04

# Or use the micro image for simple tests
act -P ubuntu-latest=node:16-alpine
```

#### 3. Missing Tools in Container
```bash
# If tools are missing, use a fuller image
act -P ubuntu-latest=catthehacker/ubuntu:full-20.04
```

#### 4. Secrets Not Working
```bash
# Make sure secrets file has correct format
echo "SECRET_NAME=secret_value" > .secrets
act --secret-file .secrets
```

### Debugging Tips

#### Verbose Output
```bash
# Get detailed logs
act -v

# Get even more detailed logs
act -vv
```

#### Interactive Debugging
```bash
# Drop into shell if job fails
act --shell

# Keep containers running after completion
act --reuse
```

#### Check What Would Run
```bash
# List all jobs that would run
act -l

# List jobs for specific event
act pull_request -l
```

## Best Practices

### 1. Create Act Configuration
Create `.actrc` file in your project root:

```bash
# .actrc
-P ubuntu-latest=catthehacker/ubuntu:act-latest
--container-daemon-socket -
```

### 2. Use .gitignore for Act Files
Add to your `.gitignore`:

```gitignore
# Act testing files
.secrets
.env
*-event.json
```

### 3. Test Before Push
```bash
# Always test your workflows before pushing
act pull_request -W .github/workflows/pr.yml
git add .
git commit -m "feat: add new feature"
git push
```

### 4. Use Make Targets for Act
Add to your `Makefile`:

```makefile
# Test GitHub Actions locally
test-actions:
	act pull_request -W .github/workflows/pr.yml
	act push -W .github/workflows/go.yml

test-actions-verbose:
	act pull_request -W .github/workflows/pr.yml -v
```

## Integration with Development Workflow

### Pre-commit Testing
```bash
#!/bin/bash
# .git/hooks/pre-push
echo "Testing GitHub Actions locally..."
act pull_request -W .github/workflows/pr.yml
if [ $? -ne 0 ]; then
    echo "GitHub Actions test failed. Push aborted."
    exit 1
fi
```

### VS Code Integration
Add to `.vscode/tasks.json`:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Test GitHub Actions",
            "type": "shell",
            "command": "act",
            "args": ["pull_request", "-W", ".github/workflows/pr.yml"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
```

## Resources

- [Act GitHub Repository](https://github.com/nektos/act)
- [Act Documentation](https://github.com/nektos/act#readme)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Hub - Act Runner Images](https://hub.docker.com/u/catthehacker)

## Quick Reference

```bash
# Most common commands for our project
act pull_request -W .github/workflows/pr.yml     # Test PR workflow
act push -W .github/workflows/go.yml             # Test Go CI workflow
act -l                                           # List all jobs
act -n                                           # Dry run
act -v                                           # Verbose output
act --secret-file .secrets                       # Use secrets file
act --env-file .env                              # Use environment file
```
