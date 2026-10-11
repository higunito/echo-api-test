-- あるべきスキーマ。Atlas（migrate diff）と sqlc の両方がこのファイルを読む。
-- 変更したら `mise run db:diff <name>` でマイグレーションを生成すること。

CREATE TABLE adventurers (
  id         uuid        PRIMARY KEY,
  name       text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50),
  created_at timestamptz NOT NULL DEFAULT now()
);

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
  version      integer     NOT NULL DEFAULT 1,
  CONSTRAINT quests_status_fields CHECK (
    (status = 'open'
      AND assignee_id IS NULL AND accepted_at IS NULL AND completed_at IS NULL)
    OR (status = 'accepted'
      AND assignee_id IS NOT NULL AND accepted_at IS NOT NULL AND completed_at IS NULL)
    OR (status = 'completed'
      AND assignee_id IS NOT NULL AND accepted_at IS NOT NULL AND completed_at IS NOT NULL)
  )
);

CREATE INDEX quests_created_at_idx ON quests (created_at DESC, id DESC);
CREATE INDEX quests_status_created_at_idx ON quests (status, created_at DESC);
