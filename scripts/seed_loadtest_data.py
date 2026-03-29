#!/usr/bin/env python3
"""Seed load-test data for select-course project.

Creates mock users and courses with deterministic prefixes so cleanup is safe.
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
class SeedResult:
    users_inserted: int
    courses_inserted: int
    user_min_id: int
    user_max_id: int
    course_min_id: int
    course_max_id: int


def resolve_dsn(cli_dsn: str | None) -> str:
    if cli_dsn:
        return cli_dsn

    env_dsn = os.getenv("DB_DSN") or os.getenv("GOOSE_DBSTRING")
    if env_dsn:
        return env_dsn

    return "host=localhost port=5432 user=postgres password=postgres dbname=select-course sslmode=disable"


def ensure_category_exists(conn: psycopg.Connection, category_id: int) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT id
            FROM course_categories
            WHERE id = %s
            LIMIT 1
            """,
            (category_id,),
        )
        row = cur.fetchone()
        if not row:
            raise ValueError(f"category_id {category_id} does not exist in course_categories")


def reset_tables_and_sequences(conn: psycopg.Connection) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            TRUNCATE TABLE user_courses, users, courses
            RESTART IDENTITY
            """
        )


def seed_users(conn: psycopg.Connection, count: int) -> int:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO users (username, password, flag)
            SELECT
                %s || LPAD(gs::text, 4, '0'),
                'loadtest_password',
                0
            FROM generate_series(1, %s) AS gs
            ON CONFLICT (username) DO NOTHING
            """,
            (USER_PREFIX, count),
        )
        return cur.rowcount if cur.rowcount != -1 else 0


def seed_courses(conn: psycopg.Connection, count: int, category_id: int, capacity: int) -> int:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO courses (title, category_id, week, duration, capacity)
            SELECT
                %s || LPAD(gs::text, 4, '0'),
                %s,
                ((gs - 1) %% 5)::int,
                ((gs - 1) %% 3)::text,
                %s
            FROM generate_series(1, %s) AS gs
            """,
            (COURSE_PREFIX, category_id, capacity, count),
        )
        return cur.rowcount if cur.rowcount != -1 else 0


def get_seeded_user_range(conn: psycopg.Connection) -> tuple[int, int]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT MIN(id), MAX(id)
            FROM users
            WHERE username LIKE %s
            """,
            (f"{USER_PREFIX}%",),
        )
        row = cur.fetchone()
        if not row or row[0] is None or row[1] is None:
            raise ValueError("no seeded users found")
        return int(row[0]), int(row[1])


def get_seeded_course_range(conn: psycopg.Connection) -> tuple[int, int]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT MIN(id), MAX(id)
            FROM courses
            WHERE title LIKE %s
            """,
            (f"{COURSE_PREFIX}%",),
        )
        row = cur.fetchone()
        if not row or row[0] is None or row[1] is None:
            raise ValueError("no seeded courses found")
        return int(row[0]), int(row[1])


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Seed users and courses for load testing")
    parser.add_argument("--dsn", help="PostgreSQL DSN. Defaults to DB_DSN/GOOSE_DBSTRING/.env style fallback")
    parser.add_argument("--user-count", type=int, default=500, help="Number of users to create (default: 50)")
    parser.add_argument("--course-count", type=int, default=50, help="Number of courses to create (default: 50)")
    parser.add_argument("--category-id", type=int, default=1, help="Existing category_id for new courses (default: 1)")
    parser.add_argument("--capacity", type=int, default=20, help="Default capacity per generated course (default: 100)")
    return parser.parse_args()


def run() -> int:
    args = parse_args()

    if args.user_count <= 0:
        print("--user-count must be > 0", file=sys.stderr)
        return 1

    if args.course_count <= 0:
        print("--course-count must be > 0", file=sys.stderr)
        return 1

    if args.category_id <= 0:
        print("--category-id must be > 0", file=sys.stderr)
        return 1

    if args.capacity <= 0:
        print("--capacity must be > 0", file=sys.stderr)
        return 1

    dsn = resolve_dsn(args.dsn)

    try:
        with psycopg.connect(dsn) as conn:
            with conn.transaction():
                ensure_category_exists(conn, args.category_id)
                reset_tables_and_sequences(conn)
                users_inserted = seed_users(conn, args.user_count)
                courses_inserted = seed_courses(conn, args.course_count, args.category_id, args.capacity)
                user_min_id, user_max_id = get_seeded_user_range(conn)
                course_min_id, course_max_id = get_seeded_course_range(conn)

            result = SeedResult(
                users_inserted=users_inserted,
                courses_inserted=courses_inserted,
                user_min_id=user_min_id,
                user_max_id=user_max_id,
                course_min_id=course_min_id,
                course_max_id=course_max_id,
            )
    except Exception as exc:
        print(f"Failed to seed load-test data: {exc}", file=sys.stderr)
        return 1

    print("Seed completed")
    print(f"  users inserted  : {result.users_inserted}")
    print(f"  courses inserted: {result.courses_inserted}")
    print(f"  category_id     : {args.category_id}")
    print(f"  user id range   : {result.user_min_id} ~ {result.user_max_id}")
    print(f"  course id range : {result.course_min_id} ~ {result.course_max_id}")
    print(f"  user prefix     : {USER_PREFIX}")
    print(f"  course prefix   : {COURSE_PREFIX}")
    print("  k6 env hint     :")
    print(f"    USER_START_ID={result.user_min_id}")
    print(f"    USER_COUNT={args.user_count}")
    print(f"    COURSE_START_ID={result.course_min_id}")
    print(f"    COURSE_COUNT={args.course_count}")
    return 0


if __name__ == "__main__":
    raise SystemExit(run())
