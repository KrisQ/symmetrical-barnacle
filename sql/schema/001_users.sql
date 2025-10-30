-- +goose Up
create table users (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    "name" text not null
);

-- +goose Down
drop table users;
