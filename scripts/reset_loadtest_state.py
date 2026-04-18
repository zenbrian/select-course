#!/usr/bin/env python3
"""Reset load-test data: clear user_courses and restore user flags + course capacities.

Useful between k6 scenarios — keeps users and courses but resets all selection state.
"""

from __future__ import annotations

import argparse
import os
import sys

try:
    import psycopg
except ImportError as exc:
    print("Missing dependency: psycopg. Install with: pip install 'psycopg[binary]'", file=sys.stderr)
    raise SystemExit(1) from exc


USER_PREFIX = "lt_user_"
COURSE_PREFIX = "lt_course_"


def resolve_dsn(cli_dsn: str | None) -> str:
    if cli_dsn:
        return cli_dsn
    env_dsn = os.getenv("DB_DSN") or os.getenv("GOOSE_DBSTRING")
    if env_dsn:
        return env_dsn
    return "host=localhost port=5432 user=postgres password=postgres dbname=select-course sslmode=disable"


def reset(conn: psycopg.Connection, capacity: int) -> dict:
    with conn.cursor() as cur:
        # 1. Delete all user_courses
        cur.execute("DELETE FROM user_courses")
        uc_deleted = cur.rowcount if cur.rowcount != -1 else 0

        # 2. Reset user flags to 0
        cur.execute(
            "UPDATE users SET flag = 0 WHERE username LIKE %s",
            (f"{USER_PREFIX}%",),
        )
        users_reset = cur.rowcount if cur.rowcount != -1 else 0

        # 3. Restore course capacities to given value
        cur.execute(
            "UPDATE courses SET capacity = %s WHERE title LIKE %s",
            (capacity, f"{COURSE_PREFIX}%"),
        )
        courses_reset = cur.rowcount if cur.rowcount != -1 else 0

    return {
        "uc_deleted": uc_deleted,
        "users_reset": users_reset,
        "courses_reset": courses_reset,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Reset load-test state (keep users/courses, clear selections)")
    parser.add_argument("--dsn", help="PostgreSQL DSN")
    parser.add_argument("--capacity", type=int, default=20, help="Restore course capacity to this value (default: 20)")
    return parser.parse_args()


def run() -> int:
    args = parse_args()
    dsn = resolve_dsn(args.dsn)

    try:
        with psycopg.connect(dsn) as conn:
            with conn.transaction():
                result = reset(conn, args.capacity)
    except Exception as exc:
        print(f"Failed to reset: {exc}", file=sys.stderr)
        return 1

    print("Reset completed")
    print(f"  user_courses deleted : {result['uc_deleted']}")
    print(f"  user flags reset    : {result['users_reset']}")
    print(f"  course caps restored: {result['courses_reset']} (capacity={args.capacity})")
    return 0


if __name__ == "__main__":
    raise SystemExit(run())
