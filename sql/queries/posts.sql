-- name: CreatePost :one
insert into posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
values (
  $1, $2, $3, $4, $5, $6, $7, $8
)
returning *;

-- name: GetPostsForUser :many
select posts.*, feeds.name as feed_name
from posts
inner join feeds on posts.feed_id = feeds.id
inner join feed_follows on feeds.id = feed_follows.feed_id
where feed_follows.user_id = $1
order by posts.published_at desc nulls last, posts.created_at desc
limit $2;

