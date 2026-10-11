-- Create "adventurers" table
CREATE TABLE "adventurers" (
  "id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "adventurers_name_check" CHECK ((char_length(name) >= 1) AND (char_length(name) <= 50))
);
-- Create "quests" table
CREATE TABLE "quests" (
  "id" uuid NOT NULL,
  "title" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "reward" integer NOT NULL,
  "deadline" timestamptz NULL,
  "status" text NOT NULL,
  "assignee_id" uuid NULL,
  "accepted_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "version" integer NOT NULL DEFAULT 1,
  PRIMARY KEY ("id"),
  CONSTRAINT "quests_assignee_id_fkey" FOREIGN KEY ("assignee_id") REFERENCES "adventurers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "quests_description_check" CHECK (char_length(description) <= 2000),
  CONSTRAINT "quests_reward_check" CHECK (reward >= 0),
  CONSTRAINT "quests_status_check" CHECK (status = ANY (ARRAY['open'::text, 'accepted'::text, 'completed'::text])),
  CONSTRAINT "quests_status_fields" CHECK (((status = 'open'::text) AND (assignee_id IS NULL) AND (accepted_at IS NULL) AND (completed_at IS NULL)) OR ((status = 'accepted'::text) AND (assignee_id IS NOT NULL) AND (accepted_at IS NOT NULL) AND (completed_at IS NULL)) OR ((status = 'completed'::text) AND (assignee_id IS NOT NULL) AND (accepted_at IS NOT NULL) AND (completed_at IS NOT NULL))),
  CONSTRAINT "quests_title_check" CHECK ((char_length(title) >= 1) AND (char_length(title) <= 100))
);
-- Create index "quests_created_at_idx" to table: "quests"
CREATE INDEX "quests_created_at_idx" ON "quests" ("created_at" DESC, "id" DESC);
-- Create index "quests_status_created_at_idx" to table: "quests"
CREATE INDEX "quests_status_created_at_idx" ON "quests" ("status", "created_at" DESC);
