-- クエストのクエリ。`mise run gen`（sqlc generate）で
-- internal/infrastructure/postgres/sqlcgen/quests.sql.go に Go の関数が生成される。
--
-- sqlc の書き方:
--   -- name: 関数名 :種類   … 直後の SQL から Go の関数を作る。種類は
--                             :one（1 行返す）/ :many（複数行）/ :exec（結果なし）/ :execrows（更新件数を返す）
--   $1, $2 …                … 位置で指定するパラメータ。Go 側では XxxParams 構造体のフィールドになる
--   @name                    … 名前付きパラメータ（sqlc.arg('name') の省略形）。フィールド名が分かりやすくなる
--   sqlc.narg('name')        … NULL を渡せる名前付きパラメータ（Go 側は pgtype.Text など NULL 可の型になる）
-- 「-- name:」行の直前のコメントは、生成される Go 関数のドキュメントコメントになる。

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

-- 一覧の 1 ページ分を新しい順に返す（FR-006, FR-006a, FR-007b）。
-- 期限切れ = 募集中かつ期限 <= now（FR-007a）。現在時刻は DB の now() ではなく @now で受け取る
-- （テストで時刻を固定するため）。status が NULL なら状態で絞り込まず、
-- exclude_expired が false なら期限切れも含める。
-- ORDER BY の id DESC は、created_at が同じ行の順番を毎回同じにするための保険。
-- name: ListQuests :many
SELECT id, title, description, reward, deadline, status,
       assignee_id, accepted_at, completed_at, created_at, version
FROM quests
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT @exclude_expired::bool
       OR NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= @now::timestamptz))
ORDER BY created_at DESC, id DESC
LIMIT @page_limit OFFSET @page_offset;

-- ListQuests と同じ絞り込み条件で総件数を数える（総ページ数の計算用）。
-- name: CountQuests :one
SELECT count(*)
FROM quests
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT @exclude_expired::bool
       OR NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= @now::timestamptz));

-- 楽観的ロック: version が一致したときだけ更新し、version を 1 進める（research R3）。
-- 更新件数が 0 なら、読み込んだあとに他の誰かが先に更新している（= 競合）。
-- name: UpdateQuest :execrows
UPDATE quests
SET status       = @status,
    assignee_id  = @assignee_id,
    accepted_at  = @accepted_at,
    completed_at = @completed_at,
    version      = version + 1
WHERE id = @id
  AND version = @version;
