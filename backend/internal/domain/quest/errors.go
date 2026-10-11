package quest

import "errors"

// ドメインエラー（data-model.md の「ドメインエラー一覧」）。
//
// Go では「センチネルエラー」と呼ばれる、パッケージレベルの変数としてエラーを定義する書き方がよく使われる。
// 呼び出し側は `errors.Is(err, quest.ErrQuestNotFound)` のように比較して、どのエラーかを判定する。
// errors.Is は fmt.Errorf("...: %w", err) で包まれた（ラップされた）エラーの中まで調べてくれるので、
// 途中の層で文脈を足しても判定は壊れない。
//
// HTTP ステータスや利用者向けメッセージへの変換は handler 層（handler/errors.go）が行う。
// domain は「何が起きたか」だけを表し、「どう見せるか」は知らない。
var (
	// 入力の検証（NewQuest）
	ErrInvalidTitle       = errors.New("quest: title must be 1-100 characters")
	ErrInvalidDescription = errors.New("quest: description must be at most 2000 characters")
	ErrInvalidReward      = errors.New("quest: reward must be 0 or greater")
	ErrDeadlineInPast     = errors.New("quest: deadline must be in the future")

	// 取得（Repository）
	ErrQuestNotFound = errors.New("quest: not found")

	// 状態遷移（Accept / Complete）
	ErrQuestNotOpen          = errors.New("quest: not open")
	ErrQuestExpired          = errors.New("quest: expired")
	ErrQuestNotAccepted      = errors.New("quest: not accepted")
	ErrQuestAlreadyCompleted = errors.New("quest: already completed")
	ErrNotAssignee           = errors.New("quest: not the assignee")

	// 楽観的ロックの競合（Repository.Update）。利用者には直接見せず、usecase で上の状態エラーに読み替える。
	ErrConcurrentModification = errors.New("quest: concurrent modification")
)
