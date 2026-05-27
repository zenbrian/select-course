-- name: CreateCourse :one
INSERT INTO courses (
    title,
    category_id,
    week,
    duration,
    capacity
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at;

-- name: GetCourseByID :one
SELECT
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at
FROM courses
WHERE id = $1;

-- name: GetCourseByIDForUpdate :one
SELECT
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at
FROM courses
WHERE id = $1
FOR UPDATE;

-- name: GetUserByIDForUpdate :one
SELECT id, username, password, created_at, updated_at, flag
FROM users
WHERE id = $1
FOR UPDATE;

-- name: UpdateCourse :one
UPDATE courses
SET
    title = $2,
    category_id = $3,
    week = $4,
    duration = $5,
    capacity = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at;

-- name: TryDecrementCapacity :one
UPDATE courses
SET
    capacity = capacity - 1,
    updated_at = NOW()
WHERE id = $1
  AND capacity > 0
RETURNING
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at;

-- name: IncrementCapacity :one
UPDATE courses
SET
    capacity = capacity + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at;

-- name: DeleteCourse :exec
DELETE FROM courses
WHERE id = $1;

-- name: GetUserByID :one
SELECT id, username, password, created_at, updated_at, flag
FROM users
WHERE id = $1;

-- name: CreateUserCourse :exec
INSERT INTO user_courses (user_id, course_id)
VALUES ($1, $2);

-- name: UpdateUserFlag :exec
UPDATE users
SET flag = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteUserCourse :execrows
DELETE FROM user_courses
WHERE user_id = $1 AND course_id = $2;

-- name: ListCourses :many
SELECT
    id,
    title,
    category_id,
    week,
    duration,
    capacity,
    created_at,
    updated_at
FROM courses;

-- name: CreateUser :one
INSERT INTO users (
    username,
    password,
    flag
) VALUES (
    $1, $2, 0
)
RETURNING id, username, password, created_at, updated_at, flag;

-- name: GetUserByUsername :one
SELECT id, username, password, created_at, updated_at, flag
FROM users
WHERE username = $1;

-- name: ListUsers :many
SELECT id, username, password, created_at, updated_at, flag
FROM users;

-- name: GetCoursesByUserID :many
SELECT c.id, c.title, c.category_id, c.week, c.duration, c.capacity, c.created_at, c.updated_at
FROM courses c
JOIN user_courses uc ON c.id = uc.course_id
WHERE uc.user_id = $1;