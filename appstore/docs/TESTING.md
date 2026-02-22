# Testing Guidelines

## General Principles

- Write tests for all exported functions
- Use table-driven tests when testing multiple cases
- Mock external API calls
- Test error cases, not just happy paths
- Prefer test-driven development (write tests first, then implement)

## Coverage Requirements

For each client endpoint, cover:
1. Success path
2. Validation errors
3. API error responses

## Test Patterns

### Table-Driven Tests

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "foo", "bar", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Something(tt.input)
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

### Helper Functions

- Use `t.Helper()` in test helper functions
- For JSON assertions, unmarshal and assert fields (not `strings.Contains`)

### CLI Tests

- Add CLI-level tests for command output/parsing
- Tests should capture stderr for usage text (help output goes to stderr)

## Running Tests

```bash
make test       # Run all tests
go test -v ./...  # Verbose output
go test -run TestName ./pkg  # Run specific test
```
