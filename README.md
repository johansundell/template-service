# template-service

A robust Go-based service template designed for quick bootstrapping of web services. It includes built-in support for system service management, database integration (SQLite/MySQL), authentication, and Docker deployment.

## API Endpoints

### Public Endpoints

- **GET /**
  - Health check endpoint. Returns 200 OK if the service is running.

- **GET /ping/:argument**
  - Echo endpoint. Returns `{"ping": ":argument"}`.

### Protected Endpoints

These endpoints require an `Authorization` header with the configured `AUTH_TOKEN` (e.g., `Authorization: Bearer <token>` or `Authorization: <token>`).

- **GET /pong/:argument**
  - Echo endpoint. Returns `{"pong": ":argument"}`.

- **GET /logs/:from/:to**
  - Retrieve usage logs within a date range.
  - `:from` and `:to` should be valid date strings.

## Service Management

The application can be installed as a system service.

```bash
# Install the service
./template-service -service install

# Start the service
./template-service -service start

# Stop the service
./template-service -service stop

# Uninstall the service
./template-service -service uninstall
```

## Features

- **Web Server**: Built with [Gin](https://github.com/gin-gonic/gin) for high performance.
- **Service Management**: Can be installed and managed as a system service (Windows Service, Systemd, etc.) using [kardianos/service](https://github.com/kardianos/service).
- **Database Support**: Integrated support for SQLite and MySQL.
- **Authentication**: Simple token-based authentication for protected routes.
- **Docker Ready**: Includes `Dockerfile` and `docker-compose.yml` for easy containerization.
- **Asset Management**: Supports embedding assets or serving from the file system.
- **Logging**: Request logging to database.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.24 or higher
- [Make](https://www.gnu.org/software/make/) (optional, for build scripts)
- [Docker](https://www.docker.com/) (optional, for containerized run)

### Installation

Clone the repository:

```bash
git clone https://github.com/johansundell/template-service.git
cd template-service
```

### Running Locally

You can run the service directly using Go:

```bash
go run .
```

Or build it using Make:

```bash
make build
./template-service
```

### Running with Docker

To run the service using Docker Compose:

```bash
docker-compose up --build
```

This will start the service on the configured port (default 8080).

## Configuration

The application is configured via environment variables. You can set these in a `.env` file in the root directory.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DEBUG` | bool | `false` | Enable debug mode. |
| `PORT` | string | `:8080` | The port the server listens on. |
| `USE_FILE_SYSTEM` | bool | `false` | If true, serves assets from the `assets` folder. If false, uses embedded assets. |
| `TIMEOUT` | int | `15` | Request timeout in seconds. |
| `USE_MYSQL` | bool | `false` | Enable MySQL database support. |
| `USE_SQLITE` | bool | `false` | Enable SQLite database support. |
| `AUTH_TOKEN` | string | - | Token required for protected endpoints. |
| `MYSQL_USERNAME` | string | - | MySQL username. |
| `MYSQL_PASSWORD` | string | - | MySQL password. |
| `MYSQL_HOST` | string | - | MySQL host address. |
| `MYSQL_PORT` | string | - | MySQL port. |
| `MYSQL_DATABASE` | string | - | MySQL database name. |


