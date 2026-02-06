# Contributing to tfskel

First off, thank you for considering contributing to tfskel! It's people like you that make tfskel such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples**
- **Describe the behavior you observed and what you expected**
- **Include screenshots if relevant**
- **Include your environment details** (OS, Go version, tfskel version)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **Use a clear and descriptive title**
- **Provide a detailed description of the suggested enhancement**
- **Explain why this enhancement would be useful**
- **List any similar features in other tools**

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing style
6. Write a good commit message

## Development Process

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/tfskel.git
cd tfskel

# Add upstream remote
git remote add upstream https://github.com/ishuar/tfskel.git

# Install dependencies
go mod download

# Run tests
go test ./...
```

### Installing Development Tools

```bash
# Install goimports (better formatting + import management)
go install golang.org/x/tools/cmd/goimports@latest

# Install golangci-lint (linter)
brew install golangci-lint  # macOS
# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Optional: Install security scanners
go install github.com/securego/gosec/v2/cmd/gosec@latest
brew install trivy  # macOS
```

### Makefile Commands

```bash
make build          # Build the binary
make test           # Run tests
make check          # Run all checks (fmt, vet, lint, test)
make ci             # Full CI pipeline (check + security scans)
make install        # Install binary to $GOPATH/bin
make help           # Show all available commands
```

### Development Workflow

1. **Create a branch**
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes**
   - Write clear, concise code
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**
   ```bash
   # Run all tests
   go test ./...

   # Run tests with coverage
   go test -cover ./...

   # Run linter
   golangci-lint run
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

   We use [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions or changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks
   - `style:` - Code style changes

5. **Push to your fork**
   ```bash
   git push origin feature/my-new-feature
   ```

6. **Open a Pull Request**

## Code Style Guidelines

### Go Best Practices

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Keep functions small and focused (max 50 lines)
- Use descriptive variable names
- Write godoc comments for exported functions
- Prefer composition over inheritance
- Always check and handle errors

### Testing Requirements

- Minimum 80% code coverage for new code
- Unit tests for all public functions
- Table-driven tests for functions with multiple scenarios
- Use testify/assert for assertions
- Mock external dependencies using interfaces

Example test structure:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Documentation

- Update README.md for user-facing changes
- Add godoc comments for all exported APIs
- Include code examples in documentation
- Document all CLI flags and commands
- Update CHANGELOG.md

### File Structure

Place files in the appropriate directory:

```
cmd/        - CLI commands
internal/
  app/      - High-level orchestration
  config/   - Configuration handling
  fs/       - Filesystem abstractions
  logger/   - Logging utilities
  templates/- Template rendering
  util/     - Utility functions
```

## Project-Specific Guidelines

### Adding New Templates

1. Create template file in `internal/templates/`
2. Add template to `NewRenderer()` in `renderer.go`
3. Add test in `renderer_test.go`
4. Update documentation

### Adding New Commands

1. Create command file in `cmd/`
2. Register command in `init()` function
3. Add tests
4. Update README.md with usage examples

### Modifying Configuration

1. Update `Config` struct in `internal/config/config.go`
2. Update validation logic
3. Add tests
4. Update `.tfskel.yaml` example in README

## Testing

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/config

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...

# Run specific test
go test -run TestMyFunction ./internal/config
```

### Writing Tests

- Test both success and failure cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test edge cases
- Use descriptive test names

## Pull Request Process

1. **Ensure all tests pass** locally
2. **Update documentation** as needed
3. **Follow commit message conventions**
4. **Reference relevant issues** in PR description
5. **Wait for review** - be responsive to feedback
6. **Squash commits** if requested
7. **Keep PRs focused** - one feature/fix per PR

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe tests you've added or run

## Checklist
- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] Changelog updated
```

## Questions?

Feel free to:
- Open an issue with the `question` label
- Start a discussion in GitHub Discussions
- Reach out to maintainers

## Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- Project documentation

Thank you for contributing to tfskel! 🎉
