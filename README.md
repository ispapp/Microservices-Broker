# Microservices Broker

A simple microservices broker application.

## Repository

[Microservices Broker GitHub Repository](git@github.com:ispapp/Microservices-Broker.git)

## Installation

1. Clone the repository:
```bash
git clone git@github.com:ispapp/Microservices-Broker.git
```

2. Install the dependencies:
```bash
cd Microservices-Broker
go mod tidy
```

3. Run the application:
```bash
go run main.go serve --port <your_port>
```

## Flags
- `--input, -i`: Input db folder (default: broker.db)
- `--port, -p`: Port to serve on (default: 50011)
