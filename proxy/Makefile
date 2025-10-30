.PHONY: test test-coverage test-race test-verbose bench clean

# Run all tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...
	@echo "\nGenerating detailed coverage report..."
	go test -coverprofile=coverage.out ./...
	@echo "Coverage report saved to coverage.out"
	@echo "To view HTML report, run: make coverage-html"

# View coverage report in browser
coverage-html: test-coverage
	go tool cover -html=coverage.out

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run security-critical tests only
test-security:
	@echo "Running security-critical tests..."
	@echo "Testing query builders (SQL injection prevention)..."
	go test -v ./internal/db -run TestSQLInjection
	go test -v ./internal/db -run TestValidateAndGetSortClause
	@echo "\nTesting input validation..."
	go test -v ./internal/utils -run TestValidateServerID
	go test -v ./internal/utils -run TestValidateStruct
	@echo "\nTesting authentication..."
	go test -v ./internal/middleware -run TestAPIKeyAuth

# Clean test cache and coverage files
clean:
	@echo "Cleaning test cache and coverage files..."
	go clean -testcache
	rm -f coverage.out
	@echo "Done!"

# Run all quality checks (tests + race detector + coverage)
check: test-race test-coverage
	@echo "\n✅ All quality checks passed!"

# Quick test (no race detector)
quick:
	@echo "Running quick tests..."
	go test -short ./...

# Help
help:
	@echo "Available commands:"
	@echo "  make test              - Run all tests"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make coverage-html     - View coverage report in browser"
	@echo "  make test-race         - Run tests with race detector"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-security     - Run security-critical tests only"
	@echo "  make bench             - Run benchmarks"
	@echo "  make check             - Run all quality checks"
	@echo "  make quick             - Run quick tests (no race detector)"
	@echo "  make clean             - Clean test cache and coverage files"
	@echo "  make help              - Show this help message"
