# Database Benchmarking Guide

This document describes how to run comprehensive benchmarks comparing Redis, PostgreSQL,
and SQLite for the PixelBattle application.

## Overview

The benchmarking system consists of:

1. **Database Adapters**: PostgreSQL and SQLite implementations matching Redis functionality
2. **Database-Level Benchmarks**: Direct database performance tests via a Go binary
3. **Application-Level Stress Tests**: k6 WebSocket load tests (light / medium / heavy)
4. **Metrics Collection**: Prometheus + Grafana for live visualization
5. **Automation**: Ansible role under [`benchmark/`](benchmark/README.md)

## Prerequisites

- Docker and Docker Compose
- Go 1.22+
- [k6](https://k6.io/docs/get-started/installation/)
- Python 3.11+ (for the venv and reporting scripts)

## Quick start

### 1. Generate SSL certificates (if needed)

```bash
./scripts/setup/generate-certs.sh
```

### 2. Configure environment

Create or update `pb_backend/app.env`:

```env
# Storage selection (redis, postgres, or sqlite)
STORAGE_TYPE=redis

REDIS_ADDR=redis:6379
REDIS_PSW=redis
REDIS_HISTORY=0

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=pixelbattle
POSTGRES_PASSWORD=pixelbattle
POSTGRES_DB=pixelbattle

SQLITE_PATH=./sqlite/pixelbattle.db

CANVAS_HEIGHT=250
CANVAS_WIDTH=500
```

### 3. Start the infrastructure

```bash
docker compose up -d
```

Starts: Redis, PostgreSQL, MongoDB, backend, Prometheus, Grafana, nginx.

### 4. Set up the Python venv

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

**Activate the project-wide venv at the start of every benchmarking session:**

```bash
source .venv/bin/activate
```

### 5. Run the full benchmark suite

```bash
cd benchmark
ansible-playbook playbooks/full_benchmark_suite.yml
```

This runs the complete pipeline: database setup → Go benchmarks → k6 load tests →
result collection → Markdown report + graphs.

For individual steps and advanced usage see [benchmark/README.md](benchmark/README.md).

## Benchmark scenarios

### Database-level benchmarks

| Scenario | Description |
|----------|-------------|
| Sequential write | Write 125,000 pixels sequentially (one per coordinate) |
| Concurrent write | 10 concurrent goroutines writing pixels |
| Full canvas read | `GetCanvas` performance (latest state of all pixels) |
| Mixed workload | 80 % writes, 20 % reads |
| History growth | Performance as pixel history accumulates |

### k6 stress tests

| Tier | VUs | Duration | Message rate |
|------|-----|----------|-------------|
| Light | 50 | 5 min | ~10 msg/min per VU |
| Medium | 200 | 10 min | ~20 msg/min per VU |
| Heavy | 500 | 15 min | ~30 msg/min per VU |

## Metrics

### Prometheus metrics exposed by the backend

- `database_operation_duration_seconds` — DB operation latencies
- `database_operation_total` — Total DB operations
- `database_connection_pool_size` — Connection pool metrics
- `websocket_message_duration_seconds` — WebSocket processing time
- `current_users` — Active user count
- `requests_per_second` — HTTP request rate

### Grafana dashboards

Access Grafana at <http://localhost:3000> (admin/admin):

- **Database Performance** — DB operation metrics
- **Application Performance** — Application-level metrics
- **Database Comparison** — Side-by-side comparison

## Manual execution (without Ansible)

### Database-level benchmark

```bash
cd pb_backend
go run ./cmd/benchmark/main.go \
    -storage=redis \
    -output=results.json \
    -height=250 \
    -width=500 \
    -writers=10 \
    -history=1
```

### k6 test

```bash
k6 run benchmark/k6/test-light.js
```

### Python reporting (venv must be active)

```bash
source .venv/bin/activate
python benchmark/reporting/generate-report.py results/
python benchmark/reporting/plot-graphs.py results/
```

## Results structure

```
results/
  benchmarks/
    redis_benchmark_<epoch>.json
    postgres_benchmark_<epoch>.json
    sqlite_benchmark_<epoch>.json
  k6/
    redis_light_<epoch>.json
    redis_light_summary_<epoch>.json
    …
  collected_<epoch>/
    benchmarks/
    k6/
    prometheus/
    logs/
  graphs/
    throughput_comparison.png
    latency_comparison.png
  comparison_report_<timestamp>.md
```

## Database schema

All three databases use an append-only schema that preserves the **full canvas history**:

- **Redis**: Lists per coordinate key (`pixel:y:x`)
- **PostgreSQL**: Single table with indexes for latest-pixel lookup
- **SQLite**: Single table with WAL mode and indexes

Every pixel change is stored; no data is overwritten.

## Troubleshooting

### PostgreSQL connection issues

```bash
docker compose ps postgres
docker compose logs postgres
```

### SQLite permission issues

```bash
mkdir -p sqlite && chmod 755 sqlite
```

### Prometheus not scraping

Check targets: <http://localhost:9090/targets>  
Check metrics endpoint: <http://localhost:8080/metrics>

### k6 not found

See the [k6 installation guide](https://k6.io/docs/get-started/installation/).

### Ansible / Python issues

Ensure the project-wide venv is active before running playbooks:

```bash
source .venv/bin/activate
which ansible-playbook   # should point into .venv/bin/
```
