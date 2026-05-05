# Benchmarks and load testing

Everything related to database benchmarks, k6 WebSocket load tests, result collection,
and Markdown/plot reporting lives under **`benchmark/`**.

## Directory layout

```
benchmark/
  .ansible-lint           # ansible-lint rule configuration
  ansible.cfg             # Ansible configuration (inventory, roles_path, callbacks)
  inventory.yml           # Localhost inventory pointing at venv Python
  requirements.txt        # References project-wide requirements.txt at repo root
  k6/
    test-light.js         # 50 VUs, 5 min
    test-medium.js        # 200 VUs, 10 min
    test-heavy.js         # 500 VUs, 15 min
  playbooks/
    full_benchmark_suite.yml   # Runs the complete benchmark pipeline
    setup_database.yml
    run_db_benchmarks.yml
    run_k6_tests.yml
    collect_results.yml
    generate_report.yml
  reporting/
    generate-report.py    # Produces a Markdown comparison report from results/
    plot-graphs.py        # Produces throughput and latency PNG graphs from results/
  roles/
    benchmark/
      defaults/main.yml   # All default variables (canvas size, storage types, paths, …)
      meta/main.yml       # Role metadata
      tasks/
        main.yml          # Full-suite entry point (called by full_benchmark_suite.yml)
        setup_database.yml
        run_db_benchmarks.yml
        run_k6_tests.yml
        collect_results.yml
        generate_report.yml
```

Artifacts are written to **`../results/`** (repository root), which is gitignored.

## Prerequisites

- Docker and Docker Compose (app + Prometheus + databases)
- Go 1.22+ (database benchmark binary)
- [k6](https://k6.io/docs/get-started/installation/) (WebSocket load tests)
- Python 3.11+ (venv; see setup below)

## Python venv setup

The project uses a **single project-wide venv** at the repository root.
Run once from the **project root**:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

The benchmark role's `benchmark_python_interpreter` variable points at
`.venv/bin/python` in the project root, so Ansible tasks that call the
reporting scripts automatically use the venv.

**Activate the venv before every session:**

```bash
source .venv/bin/activate
```

## Running benchmarks

All commands are run from the **`benchmark/`** directory (so `ansible.cfg` and
`inventory.yml` resolve without any extra flags).

### Full suite

```bash
cd benchmark
ansible-playbook playbooks/full_benchmark_suite.yml
```

Executes in order: setup → DB benchmarks → k6 tests → collect results → generate report.

### Individual playbooks

```bash
# Setup a specific database before running
ansible-playbook playbooks/setup_database.yml -e storage_type=postgres

# Database-level Go benchmarks (all storage types)
ansible-playbook playbooks/run_db_benchmarks.yml

# Override benchmark parameters
ansible-playbook playbooks/run_db_benchmarks.yml \
  -e "canvas_height=250 canvas_width=500 concurrent_writers=20"

# Single k6 test run
ansible-playbook playbooks/run_k6_tests.yml -e "storage_type=redis test_type=light"

# Collect results + container logs + Prometheus snapshot
ansible-playbook playbooks/collect_results.yml

# Custom Prometheus endpoint
ansible-playbook playbooks/collect_results.yml -e "prometheus_url=http://prometheus:9090"

# Generate report + graphs from existing results/
ansible-playbook playbooks/generate_report.yml
```

### Partial runs with tags

Every task is tagged so you can skip or target steps:

| Tag | What it runs |
|-----|-------------|
| `setup` | Database readiness checks |
| `db_bench` | Go benchmark binary |
| `k6` | k6 load tests |
| `collect` | Results + log collection |
| `report` | Markdown report + PNG graphs |

```bash
# Only run the reporting step
ansible-playbook playbooks/full_benchmark_suite.yml --tags report

# Skip collection and reporting
ansible-playbook playbooks/full_benchmark_suite.yml --skip-tags collect,report
```

## Playbook variables

All defaults live in `roles/benchmark/defaults/main.yml`. Override any of them with `-e`:

| Variable | Default | Description |
|----------|---------|-------------|
| `storage_types` | `[redis, postgres, sqlite]` | Storage backends to benchmark |
| `test_types` | `[light, medium, heavy]` | k6 test tiers to run |
| `canvas_height` | `250` | Canvas height for DB benchmarks |
| `canvas_width` | `500` | Canvas width for DB benchmarks |
| `concurrent_writers` | `10` | Goroutines for concurrent write test |
| `history_multiplier` | `1` | Writes per coordinate for history test |
| `prometheus_url` | `http://localhost:9090` | Prometheus API endpoint |
| `storage_type` | `redis` | Storage type for single-step playbooks |
| `test_type` | `light` | k6 test tier for single-step playbooks |

## Linting

With the project-wide venv active:

```bash
source .venv/bin/activate  # from project root
cd benchmark
ansible-lint
```

## Results structure

```
results/
  benchmarks/
    redis_benchmark_<epoch>.json
    postgres_benchmark_<epoch>.json
    sqlite_benchmark_<epoch>.json
    benchmark_log_<epoch>.txt
  k6/
    redis_light_<epoch>.json
    redis_light_summary_<epoch>.json
    …
  collected_<epoch>/
    benchmarks/
    k6/
    prometheus/
    logs/
    collection_summary.txt
  graphs/
    throughput_comparison.png
    latency_comparison.png
  comparison_report_<timestamp>.md
  report_generation_log_<epoch>.txt
```

## Design notes

- **Single Ansible role** (`roles/benchmark/`) contains all task files. Playbooks are
  thin wrappers that include the role or specific task files via `include_role`.
- **All defaults** are in `roles/benchmark/defaults/main.yml`; no variable is scattered
  across multiple playbooks.
- **FQCN** (`ansible.builtin.*`) is used on every module call.
- **`changed_when: false`** is set on all `command`/`shell` tasks that are read-only or
  produce idempotent side-effects (benchmark runs always create new timestamped files).
- **Tags** on every task enable partial runs without editing playbooks.
- **Block/rescue** wraps the Prometheus collection so a missing Prometheus instance does
  not abort the suite.
