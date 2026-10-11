// Package quest はクエスト（依頼）のドメインモデル。
//
// クリーンアーキテクチャの一番内側（domain 層）にあたる。
// ここにはビジネスルール（状態遷移、入力の制約、期限切れの判定）だけを置き、
// Echo（HTTP）や pgx / sqlc（DB）には一切依存しない（憲法 II）。
// そのため、このパッケージのテストは DB もサーバも起動せずに `go test` だけで動く。
//
// 依存の向き:
//
//	handler（HTTP） → usecase → domain ← infrastructure（DB）
//
// infrastructure は domain に定義された Repository インターフェースを「実装する」側なので、
// 矢印は domain に向かう。domain から外側を参照することはない。
//
// ファイル構成:
//   - quest.go      … Quest 集約（生成・状態遷移）
//   - status.go     … 状態の型（open / accepted / completed）
//   - errors.go     … ドメインエラー
//   - repository.go … 保存・取得のインターフェース（実装は infrastructure/postgres）
package quest
