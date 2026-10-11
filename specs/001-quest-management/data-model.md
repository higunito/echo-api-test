# Data Model: クエスト管理の基本機能

**Feature**: `001-quest-management` | **Date**: 2026-10-11

## エンティティ

### Adventurer（冒険者）

| 属性 | 型 | 制約 | 説明 |
|------|----|------|------|
| id | UUID | PK | 冒険者 ID |
| name | string | 1〜50 文字、必須 | 表示名 |
| created_at | timestamp (UTC) | 必須 | 作成日時 |

- 本機能では参照のみ（seed で投入）。登録・編集は別機能。

### Quest（クエスト）— 集約ルート

| 属性 | 型 | 制約 | 説明 |
|------|----|------|------|
| id | UUID | PK | クエスト ID（アプリ側で UUIDv7 を生成） |
| title | string | 1〜100 文字、必須（前後の空白は除去してから判定） | FR-002 |
| description | string | 0〜2000 文字 | 説明 |
| reward | int | 0 以上 2,147,483,647 以下 | 報酬（ゴールド）。FR-003 |
| deadline | timestamp (UTC) / null | 登録時点より未来 | 期限。任意。FR-004 |
| status | enum | `open` / `accepted` / `completed` | 募集中 / 受注済 / 完了 |
| assignee_id | UUID / null | FK → Adventurer.id | 受注者 |
| accepted_at | timestamp / null | | 受注日時 |
| completed_at | timestamp / null | | 完了日時 |
| created_at | timestamp | 必須 | 登録日時（一覧の並び順キー） |
| version | int | 必須、初期値 1 | 楽観的ロック用（research R3） |

**不変条件**（ドメインと DB の CHECK 制約の両方で守る）:

- `status = open` ⇔ `assignee_id IS NULL AND accepted_at IS NULL AND completed_at IS NULL`
- `status = accepted` ⇔ `assignee_id`・`accepted_at` が非 null、`completed_at IS NULL`
- `status = completed` ⇔ `assignee_id`・`accepted_at`・`completed_at` が非 null
- `reward >= 0`、`char_length(title) BETWEEN 1 AND 100`

**派生属性**:

- `is_expired(now)` = `status = open AND deadline IS NOT NULL AND deadline <= now`（保存しない。FR-007a）

## 状態遷移（FR-011）

```text
           Accept(adventurer, now)              Complete(adventurer, now)
  ┌──────┐ ─────────────────────────▶ ┌──────────┐ ──────────────────────────▶ ┌───────────┐
  │ open │                            │ accepted │                             │ completed │
  └──────┘                            └──────────┘                             └───────────┘
```

| 操作 | 事前条件 | 失敗時のドメインエラー |
|------|----------|------------------------|
| `Accept(adventurerID, now)` | `status = open` | `status = accepted/completed` → `ErrQuestNotOpen` |
| | 期限切れでない（`!IsExpired(now)`） | `ErrQuestExpired` |
| `Complete(adventurerID, now)` | `status = accepted` | `status = open` → `ErrQuestNotAccepted`、`completed` → `ErrQuestAlreadyCompleted` |
| | `assignee_id = adventurerID` | `ErrNotAssignee` |

- 受注済クエストは期限を過ぎても完了報告できる（spec Edge Cases）。
- 上記以外の遷移（完了 → 募集中など）を行うメソッドは存在しない。

## ドメインエラー一覧

| エラー | 発生箇所 | 意味 |
|--------|----------|------|
| `ErrInvalidTitle` | `NewQuest` | タイトルが空または 100 文字超 |
| `ErrInvalidDescription` | `NewQuest` | 説明が 2000 文字超 |
| `ErrInvalidReward` | `NewQuest` | 報酬が負 |
| `ErrDeadlineInPast` | `NewQuest` | 期限が現在時刻以前 |
| `ErrQuestNotFound` | リポジトリ | クエストが存在しない |
| `ErrAdventurerNotFound` | リポジトリ | 冒険者が存在しない |
| `ErrQuestNotOpen` | `Accept` | 既に受注済または完了 |
| `ErrQuestExpired` | `Accept` | 期限切れ |
| `ErrQuestNotAccepted` | `Complete` | 未受注 |
| `ErrQuestAlreadyCompleted` | `Complete` | 既に完了 |
| `ErrNotAssignee` | `Complete` | 受注者本人ではない |
| `ErrConcurrentModification` | リポジトリ | 楽観的ロックの競合。usecase で操作ごとに変換する: 受注（AcceptQuest）では `ErrQuestNotOpen`、完了報告（CompleteQuest）では `ErrQuestAlreadyCompleted` |

## リポジトリインターフェース（domain に定義）

- `QuestRepository`
  - `Create(ctx, *Quest) error`
  - `FindByID(ctx, id) (*Quest, error)` — 無ければ `ErrQuestNotFound`
  - `Update(ctx, *Quest) error` — `version` 一致時のみ更新し、成功時に `version` を進める。
    不一致なら `ErrConcurrentModification`
  - `List(ctx, QuestFilter, now, page) (quests []*Quest, totalCount int, err error)`
- `AdventurerRepository`
  - `FindByID(ctx, id) (*Adventurer, error)` — 無ければ `ErrAdventurerNotFound`
  - `ListAll(ctx) ([]*Adventurer, error)`

`QuestFilter` = `{ Status *Status; ExcludeExpired bool }`、ページサイズは 20 固定。

## テーブルとインデックス（`backend/db/schema.sql` の要点）

- `adventurers(id uuid PK, name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now())`
- `quests(...)` 上記属性。`status` は `text` + CHECK 制約（`open|accepted|completed`）。
  不変条件を CHECK 制約で表現。`assignee_id` は `adventurers(id)` への FK。
- インデックス: `quests(created_at DESC, id DESC)`、`quests(status, created_at DESC)`
