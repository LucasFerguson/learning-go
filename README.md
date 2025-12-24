# Go Starter Project

A simple Go project to get started with Go development.

## Prerequisites

- Go 1.22.2 or later installed on your system
- Download from: https://golang.org/dl/

## Go CLI Commands

### Running the Project

Run the program directly without creating a binary:
```bash
go run main.go
```

### Building the Project

Compile the program into an executable binary:
```bash
go build
```

This creates an executable named `hello` (or `hello.exe` on Windows) in the current directory.

Run the compiled binary:
```bash
./hello
```

Build with a custom output name:
```bash
go build -o myapp
```

### Installing the Project

Install the binary to your `$GOPATH/bin` directory:
```bash
go install
```

### Managing Dependencies

Initialize a new module (already done in this project):
```bash
go mod init example.com/hello
```

Download and add dependencies:
```bash
go get <package-name>
```

Remove unused dependencies and update go.mod:
```bash
go mod tidy
```

Download dependencies to local cache:
```bash
go mod download
```

### Testing

Run all tests in the current directory and subdirectories:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

### Formatting and Linting

Format all Go files in the current directory:
```bash
go fmt ./...
```

Check for common mistakes:
```bash
go vet ./...
```

### Other Useful Commands

View documentation for a package:
```bash
go doc fmt.Println
```

List all available Go environment variables:
```bash
go env
```

View the current Go version:
```bash
go version
```

Clean build cache:
```bash
go clean -cache
```

## Project Structure

```
.
├── go.mod          # Module definition and dependencies
└── main.go         # Main application entry point
```

## Getting Started

1. Clone or download this repository
2. Navigate to the project directory
3. Run the program:
   ```bash
   go run main.go
   ```
4. Modify `main.go` to build your own application
5. Add dependencies as needed with `go get`

## Learn More

- [Official Go Documentation](https://golang.org/doc/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Tour](https://tour.golang.org/)
