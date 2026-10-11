-- name: CreateQuest :exec
INSERT INTO quests (
  id, title, description, reward, deadline, status,
  assignee_id, accepted_at, completed_at, created_at, version
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetQuest :one
SELECT id, title, description, reward, deadline, status,
       assignee_id, accepted_at, completed_at, created_at, version
FROM quests
WHERE id = $1;

-- 期限切れ = 募集中かつ期限 <= now（FR-007a）。status / exclude_expired は省略可能な絞り込み。

-- name: ListQuests :many
SELECT id, title, description, reward, deadline, status,
       assignee_id, accepted_at, completed_at, created_at, version
FROM quests
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT @exclude_expired::bool
       OR NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= @now::timestamptz))
ORDER BY created_at DESC, id DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountQuests :one
SELECT count(*)
FROM quests
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT @exclude_expired::bool
       OR NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= @now::timestamptz));

-- 楽観的ロック: version が一致したときだけ更新する（research R3）。
-- name: UpdateQuest :execrows
UPDATE quests
SET status       = @status,
    assignee_id  = @assignee_id,
    accepted_at  = @accepted_at,
    completed_at = @completed_at,
    version      = version + 1
WHERE id = @id
  AND version = @version;
