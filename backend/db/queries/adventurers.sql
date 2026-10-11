-- 冒険者のクエリ。書き方は quests.sql の先頭コメントを参照。

-- name: GetAdventurer :one
SELECT id, name, created_at
FROM adventurers
WHERE id = $1;

-- name: ListAdventurers :many
SELECT id, name, created_at
FROM adventurers
ORDER BY name;
