#!/usr/bin/env python3
"""
Validate k6 summary JSON under results/candidate/<run-id>/k6/ for Phase 2 merge gate.

Default gates (override with CLI flags):
  * checks{stage:nominal} value >= 0.999
  * checks{stage:stress} value >= 0.99
  * min(saturation_vus) > baseline_saturation_floor (default 575)

Exit 0 if all gates pass; exit 1 otherwise.

Example:
  python3 benchmark/reporting/validate_phase2_candidate_k6.py results/candidate/1778523958
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any, Dict, Optional


def _pick_summary(k6_dir: Path, stem_prefix: str) -> Optional[Path]:
    """Latest file matching ``redis_<tier>_summary_<runid>.json`` for this directory."""
    matches = sorted(k6_dir.glob(f"{stem_prefix}_summary_*.json"), key=lambda p: p.name, reverse=True)
    return matches[0] if matches else None


def _metric_value(metrics: Dict[str, Any], key: str) -> Optional[float]:
    block = metrics.get(key)
    if not isinstance(block, dict):
        return None
    v = block.get("value")
    try:
        return float(v) if v is not None else None
    except (TypeError, ValueError):
        return None


def _saturation_min(payload: Dict[str, Any]) -> Optional[int]:
    metrics = payload.get("metrics") or {}
    if not isinstance(metrics, dict):
        return None
    sat = metrics.get("saturation_vus")
    if not isinstance(sat, dict):
        return None
    values = sat.get("values") or {}
    v = values.get("min")
    if v is None:
        v = sat.get("min")
    if v is None:
        return None
    try:
        vi = int(v)
        return vi if vi > 0 else None
    except (TypeError, ValueError):
        return None


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "run_dir",
        type=Path,
        help="Path to results/candidate/<run-id>/ (must contain k6/)",
    )
    ap.add_argument("--min-nominal", type=float, default=0.999, help="checks{stage:nominal} minimum")
    ap.add_argument("--min-stress", type=float, default=0.99, help="checks{stage:stress} minimum")
    ap.add_argument(
        "--min-saturation-vus",
        type=int,
        default=576,
        help="Minimum saturation_vus (breakpoint); default > Phase 1 baseline ~575",
    )
    ap.add_argument(
        "--skip-saturation",
        action="store_true",
        help="Do not require saturation_vus (useful while debugging breakpoint only)",
    )
    args = ap.parse_args()

    run_dir: Path = args.run_dir
    k6_dir = run_dir / "k6"
    if not k6_dir.is_dir():
        print(f"ERROR: missing k6 directory: {k6_dir}", file=sys.stderr)
        return 1

    medium = _pick_summary(k6_dir, "redis_medium")
    heavy = _pick_summary(k6_dir, "redis_heavy")
    breakpoint = _pick_summary(k6_dir, "redis_breakpoint")
    missing = [n for n, p in (("medium", medium), ("heavy", heavy), ("breakpoint", breakpoint)) if p is None]
    if missing:
        print(f"ERROR: missing k6 summaries for: {', '.join(missing)} under {k6_dir}", file=sys.stderr)
        return 1

    assert medium is not None and heavy is not None and breakpoint is not None

    nominal = json.loads(medium.read_text())
    stress = json.loads(heavy.read_text())
    brk = json.loads(breakpoint.read_text())

    ok = True
    nm = _metric_value(nominal.get("metrics") or {}, "checks{stage:nominal}")
    st = _metric_value(stress.get("metrics") or {}, "checks{stage:stress}")
    sat = _saturation_min(brk)

    print(f"Using: {medium.name}, {heavy.name}, {breakpoint.name}")
    print(f"checks{{stage:nominal}} = {nm} (min {args.min_nominal})")
    print(f"checks{{stage:stress}}  = {st} (min {args.min_stress})")
    print(f"saturation_vus min     = {sat} (min {args.min_saturation_vus}, skip={args.skip_saturation})")

    errors: list[str] = []
    if nm is None or nm < args.min_nominal:
        errors.append("FAIL: nominal stage check rate below gate")
        ok = False
    if st is None or st < args.min_stress:
        errors.append("FAIL: stress stage check rate below gate")
        ok = False
    if not args.skip_saturation:
        if sat is None or sat < args.min_saturation_vus:
            errors.append("FAIL: saturation_vus below gate (or missing)")
            ok = False

    for line in errors:
        print(line, file=sys.stderr)

    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
