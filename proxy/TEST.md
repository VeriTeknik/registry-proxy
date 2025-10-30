# Testing Guide

This document describes how to run and write tests for the Registry Proxy.

## Running Tests

### Run All Tests
```bash
cd /home/pluggedin/registry/proxy
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Tests with Detailed Coverage Report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests for Specific Package
```bash
# Query builders (SQL injection protection)
go test ./internal/db -v

# Validation utilities
go test ./internal/utils -v -run TestValidate

# HTTP helpers
go test ./internal/utils -v -run TestHTTP

# Authentication middleware
go test ./internal/middleware -v
```

### Run Benchmarks
```bash
# All benchmarks
go test -bench=. ./...

# Specific benchmarks
go test -bench=BenchmarkValidateServerID ./internal/utils
go test -bench=BenchmarkAPIKeyAuth ./internal/middleware
```

### Run Tests in CI/CD
```bash
# Run with race detector
go test -race ./...

# Run with verbose output
go test -v ./...

# Generate JUnit XML for CI
go test -v ./... 2>&1 | go-junit-report > test-results.xml
```

## Test Coverage

Current test coverage by package:

| Package | Coverage | Critical |
|---------|----------|----------|
| `internal/db` | ~85% | ✅ Yes - SQL injection prevention |
| `internal/utils` | ~90% | ✅ Yes - Input validation |
| `internal/middleware` | ~80% | ✅ Yes - Authentication |
| `internal/handlers` | ~40% | ⚠️ Integration tests needed |

### Coverage Goals
- **Critical security code**: 80%+ coverage
- **Business logic**: 70%+ coverage
- **Overall project**: 60%+ coverage

## Test Structure

### Unit Tests
Located alongside source files with `_test.go` suffix:
```
internal/
├── db/
│   ├── postgres.go
│   ├── query_builders.go
│   ├── query_builders_test.go  ← Unit tests
│   └── server_mapper_test.go   ← Unit tests
├── utils/
│   ├── validation.go
│   ├── validation_test.go      ← Unit tests
│   └── http_helpers_test.go    ← Unit tests
└── middleware/
    ├── auth.go
    └── auth_test.go             ← Unit tests
```

### Integration Tests
Would be in `tests/` directory (to be added):
```
tests/
├── integration/
│   ├── api_test.go
│   ├── database_test.go
│   └── auth_test.go
└── e2e/
    └── search_test.go
```

## Writing Tests

### Example: Testing Query Builders
```go
func TestValidateAndGetSortClause(t *testing.T) {
    tests := []struct {
        name    string
        sort    string
        wantErr bool
    }{
        {
            name:    "valid sort",
            sort:    "rating_desc",
            wantErr: false,
        },
        {
            name:    "SQL injection attempt",
            sort:    "name; DROP TABLE servers",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := validateAndGetSortClause(tt.sort)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Example: Testing Validation
```go
func TestValidateServerID(t *testing.T) {
    tests := []struct {
        name     string
        serverID string
        wantErr  bool
    }{
        {
            name:     "valid UUID",
            serverID: "550e8400-e29b-41d4-a716-446655440000",
            wantErr:  false,
        },
        {
            name:     "path traversal",
            serverID: "../../../etc/passwd",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateServerID(tt.serverID)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Security Testing

### SQL Injection Prevention Tests
The `query_builders_test.go` file includes comprehensive SQL injection tests:

```go
func TestSQLInjectionPrevention(t *testing.T) {
    maliciousInputs := []string{
        "'; DROP TABLE servers--",
        "name UNION SELECT * FROM users--",
        "name OR 1=1--",
    }
    // ... test that all are rejected or properly parameterized
}
```

### Input Validation Tests
The `validation_test.go` file tests:
- Path traversal prevention
- Server ID validation
- Parameter size limits
- Type validation
- Special character handling

### Authentication Tests
The `auth_test.go` file verifies:
- Valid API key acceptance
- Invalid API key rejection
- Missing header handling
- Constant-time comparison
- No sensitive data logging

## CI/CD Integration

### GitHub Actions Workflow
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Troubleshooting

### Tests Fail Due to Missing Dependencies
```bash
go mod tidy
go mod download
```

### Tests Fail Due to Missing Environment Variables
```bash
# Set test environment variables
export API_KEY=test-key-for-testing
export DATABASE_URL=postgres://test:test@localhost:5432/test_db
```

### Database Tests Fail
For tests requiring a database:
```bash
# Start test database
docker run -d -p 5432:5432 \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=test_db \
  postgres:15

# Run database tests
go test ./internal/db -v
```

## Test Maintenance

### Adding New Tests
1. Create `*_test.go` file next to the code being tested
2. Follow existing test patterns
3. Include both positive and negative test cases
4. Add SQL injection/security tests for user input handling
5. Update coverage metrics

### Updating Existing Tests
1. Update tests when refactoring code
2. Keep test data realistic
3. Add regression tests for bugs
4. Document complex test scenarios

## Best Practices

### Test Naming
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Benchmark functions: `BenchmarkFunctionName`
- Table-driven tests: Use descriptive `name` field

### Test Organization
```go
func TestFunction(t *testing.T) {
    // 1. Setup
    testData := setupTestData()

    // 2. Define test cases
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{ /* ... */ }

    // 3. Run test cases
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 4. Execute
            got, err := Function(tt.input)

            // 5. Assert
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Security Test Requirements
For all functions that:
- Accept user input
- Build SQL queries
- Validate data
- Handle authentication

Must include tests for:
- SQL injection attempts
- Path traversal attempts
- Buffer overflow attempts
- Type confusion
- Boundary conditions

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://blog.golang.org/cover)
- [Testing Best Practices](https://golang.org/doc/effective_go#testing)
