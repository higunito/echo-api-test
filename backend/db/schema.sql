-- あるべきスキーマ（スキーマの「正」はこのファイル 1 つ。research R4）。
-- Atlas（migrate diff）と sqlc の両方がこのファイルを読む。
--
-- 変更の流れ（Prisma の schema.prisma → migrate dev に近い）:
--   1. このファイルを編集する
--   2. `mise run db:diff <名前>` … Atlas が「今のマイグレーション履歴」と「このファイル」の差分から
--      db/migrations/ に新しい SQL を生成し、atlas.sum（改ざん検知用のハッシュ）を更新する
--   3. `mise run db:migrate`    … 開発用 DB に適用する
--   4. クエリも変えたら `mise run gen` で sqlc のコードを作り直す
-- 生成済みのマイグレーションファイルは編集しない（atlas.sum と合わなくなる）。

CREATE TABLE adventurers (
  id         uuid        PRIMARY KEY,
  name       text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50),
  created_at timestamptz NOT NULL DEFAULT now()
);

-- timestamptz はタイムゾーン付き時刻（内部的には UTC で保存される）。
-- ルールはアプリ（domain）でも検証しているが、DB の CHECK 制約でも守り、二重の防御にしている。
CREATE TABLE quests (
  id           uuid        PRIMARY KEY,
  title        text        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 100),
  description  text        NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
  reward       integer     NOT NULL CHECK (reward >= 0),
  deadline     timestamptz NULL,
  status       text        NOT NULL CHECK (status IN ('open', 'accepted', 'completed')),
  assignee_id  uuid        NULL REFERENCES adventurers (id),
  accepted_at  timestamptz NULL,
  completed_at timestamptz NULL,
  created_at   timestamptz NOT NULL,
  -- 楽観的ロック用。更新のたびに +1 する（queries/quests.sql の UpdateQuest）
  version      integer     NOT NULL DEFAULT 1,
  -- 状態と、受注者・日時の有無の組み合わせが矛盾しないことを保証する（data-model.md の不変条件）
  CONSTRAINT quests_status_fields CHECK (
    (status = 'open'
      AND assignee_id IS NULL AND accepted_at IS NULL AND completed_at IS NULL)
    OR (status = 'accepted'
      AND assignee_id IS NOT NULL AND accepted_at IS NOT NULL AND completed_at IS NULL)
    OR (status = 'completed'
      AND assignee_id IS NOT NULL AND accepted_at IS NOT NULL AND completed_at IS NOT NULL)
  )
);

-- 一覧（新しい順）と状態での絞り込みを速くするインデックス（SC-003）
CREATE INDEX quests_created_at_idx ON quests (created_at DESC, id DESC);
CREATE INDEX quests_status_created_at_idx ON quests (status, created_at DESC);
