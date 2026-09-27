# Contributing to DevProxy

Thank you for considering contributing to DevProxy! We welcome bug reports, documentation enhancements, feature proposals, and new security detection rules.

## Development Setup

### Prerequisites
- Go 1.23 or higher
- Git

### Build & Run
```bash
git clone https://github.com/Aditya-9-6/DevProxy.git
cd DevProxy
go mod download
go run ./cmd/devproxy
```

### Running Tests
Make sure all tests pass before submitting a pull request:
```bash
go test -v ./...
```

## Adding New Security Rules
New rules can be contributed in [`pkg/analysis/rules.go`](pkg/analysis/rules.go):
1. Implement the `Rule` interface:
   ```go
   type Rule interface {
       Name() string
       Evaluate(event *ringbuffer.TrafficEvent) []*Finding
   }
   ```
2. Keep pattern evaluation efficient and zero-allocation where possible.
3. Add a corresponding test case in [`pkg/analysis/rules_test.go`](pkg/analysis/rules_test.go).

## Submitting Pull Requests
1. Fork the repository and create a feature branch (`git checkout -b feat/my-new-rule`).
2. Commit your changes with descriptive commit messages (`git commit -m 'Add Stripe secret key detection'`).
3. Push to your branch (`git push origin feat/my-new-rule`).
4. Open a Pull Request on GitHub.
