# Contributing to kubectl-nuke-go

Thank you for considering contributing to this project! Your help is greatly appreciated.

## How to Contribute

- Fork the repository and create your branch from `main`.
- Use a branch name like `feat/your-feature` for new features or `fix/your-fix` for bug fixes.
- Make your changes and add tests if applicable.
- Ensure your code passes linting and builds successfully (`go test ./...`, `golint ./...`).
- Run `go mod tidy` and commit any changes to `go.mod` or `go.sum`.
- Submit a pull request with a clear description of your changes.

## Development Setup

1. Clone the repository:
   ```sh
   git clone https://github.com/codesenju/kubectl-nuke-go.git
   cd kubectl-nuke-go
   ```

2. Build the project:
   ```sh
   go build -o kubectl-nuke
   ```

3. Run tests:
   ```sh
   go test ./...
   ```

4. Test the CLI:
   ```sh
   ./kubectl-nuke --help
   ./kubectl-nuke ns --help
   ```

## Project Structure

- `main.go`: CLI entry point with Cobra commands
- `internal/kube/`: Core namespace deletion logic
- `docs/`: Documentation files
- Tests should be placed alongside the code they test

## Testing

- Add unit tests for new functionality in the `internal/kube` package
- Test CLI commands manually to ensure they work as expected
- Ensure all existing tests continue to pass

## Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small

## Code of Conduct

Please be respectful and considerate in all interactions. Harassment or abusive behavior will not be tolerated.

## Reporting Issues

If you find a bug or have a feature request, please open an issue with as much detail as possible, including:

- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Environment details (OS, Go version, kubectl version)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
