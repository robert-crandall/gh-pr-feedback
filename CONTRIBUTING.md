# Contributing to gh-pr-feedback

Thanks for your interest in contributing! This document outlines the process for contributing to this project.

## Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/robert-crandall/gh-pr-feedback.git
   cd gh-pr-feedback
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Install development tools**
   ```bash
   # golangci-lint for linting
   brew install golangci-lint  # macOS
   # or see https://golangci-lint.run/usage/install/

   # goimports for import formatting
   go install golang.org/x/tools/cmd/goimports@latest
   ```

4. **Build and install locally**
   ```bash
   make install
   ```

## Making Changes

1. **Create a branch**
   ```bash
   git checkout -b feature/my-awesome-feature
   ```

2. **Make your changes** – write code, add tests

3. **Run checks before committing**
   ```bash
   make check  # runs fmt, lint, and test
   ```

4. **Commit with a descriptive message**
   ```bash
   git commit -m "feat: add support for filtering by author"
   ```

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes with no errors
- Add tests for new functionality

## Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make coverage
```

### Test Guidelines

- Table-driven tests are preferred
- Mock external dependencies (GitHub API, Copilot CLI)
- Aim for meaningful coverage, not 100% coverage of trivial code

## Pull Request Process

1. **Ensure CI passes** – all checks must be green
2. **Update documentation** – README, code comments as needed
3. **Keep PRs focused** – one feature or fix per PR
4. **Respond to feedback** – address review comments promptly

## Commit Message Convention

We loosely follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `test:` adding or updating tests
- `refactor:` code change that neither fixes a bug nor adds a feature
- `chore:` maintenance tasks

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include reproduction steps for bugs
- Check existing issues before creating a new one

## Questions?

Feel free to open an issue or start a discussion. We're happy to help!
