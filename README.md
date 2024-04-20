# Telegram Event Server

Telegram Event Server is a microservice written in Go that provides a WebSocket endpoint for pushing Telegram events to clients. This service leverages the gotd/td library for Telegram integration and is designed with a focus on speed and simplicity.

## Features

- WebSocket endpoint at ws://123.123.123.123:8080/events
- Integration with Telegram using gotd/td
- Docker-based deployment
- CI/CD workflow for automated builds
- Utility scripts for managing the service

## Getting Started

To get a local copy up and running, follow these steps.

### Prerequisites

- Docker
- Docker Compose Plugin

### Fast installation (recommended)
Use prebuilt Docker image from GitHub Container Registry:
```sh
ghcr.io/n0rthernl1ghts/telegram-event-server:latest
```

Use provided docker-compose.yml file.


## Usage

Once the service is up and running, you can connect to the WebSocket endpoint at 123.123.123.123:8080/events to start receiving Telegram events.

## Deployment

This service is designed to be deployed using Docker and Docker Compose. A docker-compose.yml file is provided for convenience. The CI/CD workflow automatically builds a fresh Docker image based on Alpine and pushes it to ghcr.io/n0rthernl1ghts/telegram-event-server.

## Development environment

1. Check out the repository
2. Run: `bin/build dev` (Builds from Dockerfile.dev)
3. Run `bin/shell` to drop into development environment/shell.

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are greatly appreciated.

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Project Link: [https://github.com/n0rthernl1ghts/telegram-event-server](https://github.com/n0rthernl1ghts/telegram-event-server)

## Acknowledgements

- [gotd/td](https://github.com/gotd/td)
