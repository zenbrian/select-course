#!/usr/bin/env python3
"""Cleanup load-test data for select-course project.

Deletes only records created by seed script using deterministic prefixes.
"""

from __future__ import annotations

import argparse
import os
import sys
from dataclasses import dataclass

try:
    import psycopg
except ImportError as exc:
    print("Missing dependency: psycopg. Install with: pip install 'psycopg[binary]'", file=sys.stderr)
    raise SystemExit(1) from exc


USER_PREFIX = "lt_user_"
COURSE_PREFIX = "lt_course_"


@dataclass
class CleanupResult:
    user_courses_deleted: int
    courses_deleted: int
    users_deleted: int


def resolve_dsn(cli_dsn: str | None) -> str:
    if cli_dsn:
        return cli_dsn

    env_dsn = os.getenv("DB_DSN") or os.getenv("GOOSE_DBSTRING")
    if env_dsn:
        return env_dsn

    return "host=localhost port=5432 user=postgres password=postgres dbname=select-course sslmode=disable"


def cleanup(conn: psycopg.Connection) -> CleanupResult:
    with conn.cursor() as cur:
        cur.execute(
            """
            DELETE FROM user_courses
            WHERE user_id IN (
                SELECT id FROM users WHERE username LIKE %s
            )
            OR course_id IN (
                SELECT id FROM courses WHERE title LIKE %s
            )
            """,
            (f"{USER_PREFIX}%", f"{COURSE_PREFIX}%"),
        )
        user_courses_deleted = cur.rowcount if cur.rowcount != -1 else 0

        cur.execute(
            """
            DELETE FROM courses
            WHERE title LIKE %s
            """,
            (f"{COURSE_PREFIX}%",),
        )
        courses_deleted = cur.rowcount if cur.rowcount != -1 else 0

        cur.execute(
            """
            DELETE FROM users
            WHERE username LIKE %s
            """,
            (f"{USER_PREFIX}%",),
        )
        users_deleted = cur.rowcount if cur.rowcount != -1 else 0

    return CleanupResult(
        user_courses_deleted=user_courses_deleted,
        courses_deleted=courses_deleted,
        users_deleted=users_deleted,
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Cleanup load-test users and courses")
    parser.add_argument("--dsn", help="PostgreSQL DSN. Defaults to DB_DSN/GOOSE_DBSTRING/.env style fallback")
    return parser.parse_args()


def run() -> int:
    args = parse_args()
    dsn = resolve_dsn(args.dsn)

    try:
        with psycopg.connect(dsn) as conn:
            with conn.transaction():
                result = cleanup(conn)
    except Exception as exc:
        print(f"Failed to cleanup load-test data: {exc}", file=sys.stderr)
        return 1

    print("Cleanup completed")
    print(f"  user_courses deleted: {result.user_courses_deleted}")
    print(f"  courses deleted     : {result.courses_deleted}")
    print(f"  users deleted       : {result.users_deleted}")
    print(f"  user prefix         : {USER_PREFIX}")
    print(f"  course prefix       : {COURSE_PREFIX}")
    return 0


if __name__ == "__main__":
    raise SystemExit(run())
