# Contributing to go-postmark

Thanks for your interest in contributing! This guide will help you get set up and submit a great pull request.

## Prerequisites

- [Go](https://golang.org/dl/) 1.24 or later

## Building

Clone the repo and fetch dependencies:

```bash
git clone https://github.com/tjsampson/go-postmark.git
cd go-postmark
go mod download
```

Build the package to verify everything compiles cleanly:

```bash
go build ./...
```

## Running Tests

```bash
go test ./...
```

To see verbose output:

```bash
go test -v ./...
```

> **Note:** The tests use a mock HTTP server, so no real Postmark credentials are needed.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep exported types and functions documented with Go doc comments.

Run the linter before opening a PR:

```bash
go vet ./...
```

## Opening a Pull Request

1. **Fork** the repository and create a feature branch off `main`:
   ```bash
   git checkout -b feat/my-new-feature
   ```
2. Make your changes and **add tests** for any new behaviour.
3. Ensure all tests pass (`go test ./...`) and there are no vet warnings.
4. **Commit** with a clear, descriptive message.
5. **Push** your branch and open a PR against `main`.
6. Fill in the PR template — describe what changed and why, and note how it was tested.

A maintainer will review your PR as soon as possible. We may suggest changes before merging, so please keep an eye on the review thread.

## Reporting Issues

Found a bug or have a feature request? Please [open an issue](https://github.com/tjsampson/go-postmark/issues) with enough detail to reproduce the problem or understand the request.

## License

By contributing, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
