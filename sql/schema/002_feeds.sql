-- +goose Up
create table feeds (
    id uuid primary key,
    created_at timestamp,
    updated_at timestamp,
    name text not null,
    url text not null unique,
    user_id uuid not null,
      foreign key (user_id)
        references users(id)
        on delete cascade
);

-- +goose Down
drop table if exists feeds;
