#!/usr/bin/env python3
"""
Единая точка входа для графиков ВКР: фазы БД и сравнение baseline/candidate.

Читает JSON из каталога results/ (см. benchmark/reporting/plot_db_benchmarks.py
и plot-graphs.py). При отсутствии данных печатает подсказку и завершается с кодом 0,
чтобы не ломать необязательные шаги CI.

Использование из корня репозитория:

    python3 benchmark/reporting/generate-thesis-charts.py

Опции:

    python3 benchmark/reporting/generate-thesis-charts.py /path/to/results
"""

from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path


def main() -> int:
    root = Path(__file__).resolve().parents[2]
    results = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else (root / "results")
    graphs = results / "graphs"
    thesis_dir = graphs / "thesis"
    thesis_dir.mkdir(parents=True, exist_ok=True)

    py = sys.executable
    chk = subprocess.run(
        [py, "-c", "import matplotlib"],
        cwd=str(root),
        capture_output=True,
        text=True,
    )
    if chk.returncode != 0:
        print(
            "matplotlib не найден в текущем интерпретаторе; установите зависимости "
            "или используйте окружение проекта. Графики не генерировались.",
            file=sys.stderr,
        )
        print(f"Каталог для копий: {thesis_dir}")
        return 0

    plot_db = root / "benchmark" / "reporting" / "plot_db_benchmarks.py"
    plot_cmp = root / "benchmark" / "reporting" / "plot-graphs.py"

    db_ok = subprocess.run([py, str(plot_db), str(results)], cwd=str(root)).returncode == 0
    cmp_ok = subprocess.run([py, str(plot_cmp), str(results)], cwd=str(root)).returncode == 0

    # Копии с понятными именами для архива отчёта (основные файлы остаются в graphs/ и graphs/db_benchmarks/).
    pairs = [
        (graphs / "e2e_latency_by_stage.png", thesis_dir / "e2e_latency_by_stage.png"),
        (graphs / "ws_latency_by_stage.png", thesis_dir / "ws_latency_by_stage.png"),
        (graphs / "saturation_vus_comparison.png", thesis_dir / "saturation_vus_comparison.png"),
        (graphs / "storage_throughput_comparison.png", thesis_dir / "storage_throughput_comparison.png"),
        (graphs / "storage_latency_comparison.png", thesis_dir / "storage_latency_comparison.png"),
    ]
    for src, dst in pairs:
        if src.is_file():
            shutil.copy2(src, dst)

    if db_ok:
        print(f"DB phase plots: {graphs / 'db_benchmarks'}")
    else:
        print(f"Пропуск plot_db_benchmarks: нет JSON в {results / 'benchmarks'}", file=sys.stderr)
    if cmp_ok:
        print(f"Сравнение baseline/candidate: {graphs}")
    else:
        print(
            f"Пропуск plot-graphs: ожидаются {results / 'baseline'} и/или данные k6/rum.",
            file=sys.stderr,
        )
    print(f"Копии для ВКР (при наличии исходников): {thesis_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
