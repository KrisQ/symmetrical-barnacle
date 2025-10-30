-- name: CreateFeed :one
insert into feeds (id, created_at, updated_at, name, url, user_id)
values (
  $1, $2, $3, $4, $5, $6
)
returning *;

-- name: GetFeeds :many
select feeds.*, users.name as username from feeds inner join users on feeds.user_id = users.id;

-- name: GetFeedByUrl :one
select * from feeds where feeds.url = $1;

-- name: MarkFeedFetched :exec
update feeds
set updated_at = now(), last_fetched_at = now() 
where id = $1;

-- name: GetNextFeedToFetch :one
SELECT id, url, last_fetched_at
FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST, id
LIMIT 1;


