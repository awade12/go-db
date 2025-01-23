# go-dbs

A simple CLI tool for managing databases in Docker containers. Currently supports PostgreSQL with more databases coming soon.

## Installation

```bash
go install github.com/awade12/go-db@latest
```

## Requirements

- Go 1.21 or higher
- Docker installed and running

## Usage

### Easy Mode (Quick Start)
```bash
# Create a PostgreSQL database with default settings
go-dbs create postgres

# The default configuration:
# - Port: 5432
# - Username: postgres
# - Password: postgres
# - Database: postgres
```

### Custom Mode
```bash
# Create a PostgreSQL database with custom configuration
go-dbs create-custom postgres \
  --version 15 \
  --port 5433 \
  --password mypassword \
  --user myuser \
  --db mydb \
  --volume /path/to/data \
  --memory 2g \
  --cpu 0.5

# Additional options available:
# --timezone     Container timezone (default: UTC)
# --locale       Database locale (default: en_US.utf8)
# --network      Docker network to join
# --init-script  SQL script to run on initialization
# --ssl-mode     SSL mode (disable, require, verify-ca, verify-full)
# --ssl-cert     Path to SSL certificate
# --ssl-key      Path to SSL private key
# --ssl-root-cert Path to SSL root certificate
```

### Management Commands
```bash
# Start a stopped database
go-dbs start <container-name>

# Stop a running database
go-dbs stop <container-name>

# Remove a database container
go-dbs remove <container-name>
go-dbs remove <container-name> --force  # Force removal
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

[MIT](https://choosealicense.com/licenses/mit/) 