# Contributing Guidelines

First off, thank you for considering contributing to the Distributed Rate Limiter & API Gateway! 

## Local Development Setup

1. **Prerequisites:**
   - [Go 1.22+](https://golang.org/doc/install)
   - [Docker & Docker Compose](https://docs.docker.com/get-docker/)
   - [Vegeta](https://github.com/tsenart/vegeta) (for load testing)
   - [k6](https://k6.io/) (optional, can be run via Docker)

2. **Clone the Repository:**
   ```bash
   git clone https://github.com/MayankV004/distribute-rate-limiter.git
   cd distribute-rate-limiter
   ```

3. **Run the Cluster Locally:**
   We rely heavily on Docker Compose to simulate the production topology.
   ```bash
   docker compose -f deployments/docker-compose.yaml up -d --build
   ```

## Development Workflow

1. **Branching:** Create a new branch for your feature or bug fix (`feature/my-feature` or `bugfix/issue-123`).
2. **Code Style:** We strictly adhere to standard Go formatting. Run `go fmt ./...` before committing.
3. **Linting:** Ensure your code passes `golangci-lint run`.
4. **Testing:**
   Before submitting a Pull Request, ensure the load tests still pass and perform accurately:
   ```bash
   ./scratch/exp_a.sh
   ./scratch/exp_b.sh
   ```

## Submitting a Pull Request
- Provide a clear, descriptive title.
- Explain the **Why** and the **What** in the PR body.
- If your PR affects performance, please include Vegeta/k6 benchmark results comparing your branch against `main`.

## Code of Conduct
Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
