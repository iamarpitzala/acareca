# acareca

A Go backend service.

## Prerequisites

- [Go](https://golang.org/dl/) 1.25.3 or later

## Getting Started

Clone the repository and navigate to the project directory:

```bash
git clone https://github.com/iamarpitzala/acareca.git
cd acareca
```

Install dependencies:

```bash
go mod tidy
```

Run the application:

```bash
go run .
```

## Project Structure

```
backend/
├── go.mod        # Module definition and dependencies
└── README.md
```

## Development

Build the binary:

```bash
go build -o acareca .
```

Run tests:

```bash
go test ./...
```

## License

MIT
