// Package usecase はアプリケーションの操作（ユースケース）を 1 つずつ型にしたもの。
//
// クリーンアーキテクチャの usecase 層。handler（HTTP）から呼ばれ、
// domain のエンティティと Repository インターフェースを組み合わせて 1 つの操作を完成させる。
// 例: AcceptQuest = 冒険者を確認 → クエストを取得 → Quest.Accept → 保存。
//
// このパッケージも Echo や DB には依存しない。Repository は domain のインターフェースとして受け取り、
// 現在時刻も Clock インターフェースから取るので、テストではすべて偽物に差し替えられる（fake_test.go）。
//
// 各ユースケースは同じ形をしている:
//
//	type XxxQuest struct { 依存（Repository, Clock）}
//	func NewXxxQuest(依存...) *XxxQuest   // コンストラクタ（Go では New〜 という関数を作るのが慣習）
//	func (u *XxxQuest) Execute(ctx, 入力) (出力, error)
//
// 依存は main.go で組み立てて渡す（DI コンテナは使わない。research R2）。
package usecase
