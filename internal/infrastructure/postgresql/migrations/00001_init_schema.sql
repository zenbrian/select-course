-- +goose Up
-- +goose StatementBegin

-- =========================
-- users
-- =========================
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(64) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =========================
-- course_categories
-- =========================
CREATE TABLE course_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- =========================
-- courses
-- =========================
CREATE TABLE courses (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(64) NOT NULL,

    category_id BIGINT NOT NULL,
    week INT,
    duration VARCHAR(32) NOT NULL,
    capacity INT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_category
        FOREIGN KEY (category_id)
        REFERENCES course_categories(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

-- =========================
-- user_courses (選課關係)
-- =========================
CREATE TABLE user_courses (
    user_id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, course_id),

    CONSTRAINT fk_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_course
        FOREIGN KEY (course_id)
        REFERENCES courses(id)
        ON DELETE CASCADE
);

-- =========================
-- index（效能優化）
-- =========================
CREATE INDEX idx_courses_category_id ON courses(category_id);
CREATE INDEX idx_user_courses_course_id ON user_courses(course_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS user_courses;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS course_categories;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd