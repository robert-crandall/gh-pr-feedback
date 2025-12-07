.PHONY: build test lint fmt clean install

# Build the extension binary
build:
	go build -o gh-pr-feedback .

# Run tests with race detection and coverage
test:
	go test -v -race -cover ./...

# Run tests and generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run golangci-lint
lint:
	golangci-lint run ./...

# Format all Go files
fmt:
	gofmt -w .
	goimports -w .

# Clean build artifacts
clean:
	rm -f gh-pr-feedback coverage.out coverage.html

# Install as a gh extension (local development)
install: build
	gh extension install .

# Uninstall the extension
uninstall:
	gh extension remove gh-pr-feedback

# Run the demo recording (requires VHS: https://github.com/charmbracelet/vhs)
demo:
	vhs demo.tape

# Check everything before committing
check: fmt lint test
	@echo "All checks passed!"
