<!--
Sync Impact Report
- Version change: 1.1.0 → 1.1.1
- Modified principles: none (開発ワークフローのテスト通過条件を明確化。III との矛盾を解消)
- Added sections: none
- Removed sections: none
- Deferred TODOs: none
-->
# Adventurer Guild (echo-api-test) Constitution

## Core Principles

### I. 仕様駆動開発 (NON-NEGOTIABLE)
すべての機能は spec-kit のフロー（constitution → specify → clarify → plan → tasks → implement）
を経なければならない（MUST）。仕様・計画は `specs/` 配下の成果物を正とし、
それらを経ずに直接実装してはならない（MUST NOT）。
理由: 本プロジェクトは Claude Code + spec-kit による実開発の実践が目的であるため。

### II. DDD + クリーンアーキテクチャ
バックエンドは domain / usecase / interface(handler) / infrastructure の層に分け、
依存は常に外側から内側へ向かわなければならない（MUST）。domain 層は Echo・sqlc・pgx
などのフレームワーク／DB ライブラリに依存してはならない（MUST NOT）。
理由: ビジネスルールを技術詳細から分離し、テスト容易性と学習価値を確保するため。

### III. テストファースト
domain / usecase 層の振る舞いには単体テストを書かなければならない（MUST）。
テストは実装より先に書き、失敗を確認してから実装する（Red-Green-Refactor）。
外部依存（DB 等）はインターフェース越しに差し替え可能でなければならない（MUST）。

### IV. スキーマ・クエリの明示管理
DB スキーマ変更は Atlas のマイグレーションでのみ行い（MUST）、
クエリは sqlc で型安全に生成しなければならない（MUST）。生成コードは手編集してはならない
（MUST NOT）。マイグレーションファイルはバージョン管理し、`atlas.sum` の整合性を保つこと。

### V. シンプルさ
現在の仕様が要求しない抽象化・機能を追加してはならない（YAGNI, MUST NOT）。
複雑さを導入する場合は plan.md の Complexity Tracking に理由を記録しなければならない（MUST）。

## 技術スタック制約

- バックエンド: Go + Echo v4、PostgreSQL（Docker Compose で起動）。
- データアクセス: sqlc + pgx、マイグレーションは Atlas。
- フロントエンド: Next.js (App Router) + Tailwind CSS + shadcn/ui。
- ツールバージョンは mise（`mise.toml`）で管理しなければならない（MUST）。
- 上記スタックの変更は憲法の改訂（Governance 参照）を伴わなければならない（MUST）。

## 開発ワークフロー

- 機能ごとにブランチと `specs/<feature>/` を作成し、spec・plan・tasks を揃えてから実装する。
- 実装タスクの完了時には関連するテストがすべて通ること（MUST）。失敗したまま次の実装タスクへ
  進んではならない。ただし原則 III に従い先に書くテストタスクは、失敗を確認した状態で完了としてよい。
- plan.md の Constitution Check で本憲法への適合を確認しなければならない（MUST）。
- コミットは小さく保ち、`docs:` `feat:` `fix:` 等のプレフィックスを用いる（SHOULD）。

## Governance

本憲法は他のすべての開発慣行に優先する。改訂は変更内容と理由を文書化し、
依存するテンプレート／成果物との整合を確認したうえで行う。
バージョニングは Semantic Versioning に従う: 原則の削除・再定義は MAJOR、
原則やセクションの追加・実質的拡張は MINOR、文言修正等は PATCH。
すべての plan / レビューで本憲法への適合を確認し、違反は Complexity Tracking に
正当化を記録するか、是正しなければならない（MUST）。

**Version**: 1.1.1 | **Ratified**: 2026-10-11 | **Last Amended**: 2026-10-11
