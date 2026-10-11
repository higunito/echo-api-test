// Package handler は HTTP（Echo）の入り口。
//
// クリーンアーキテクチャの一番外側（interface 層）。役割は 3 つだけ:
//  1. リクエストを読む（JSON・クエリ・パスパラメータを Go の値に変換し、形式を検証する）
//  2. usecase を呼ぶ
//  3. 結果やエラーを JSON レスポンスにする（エラーの変換は errors.go に集約）
//
// ビジネスルール（「受注者本人しか完了報告できない」など）はここには書かず、domain に任せる。
//
// 1 リクエストの流れ（例: POST /api/quests/:id/accept）:
//
//	Echo のルータ（router.go）
//	  → ミドルウェア（RequestID・Recover・ログ）
//	  → QuestHandler.action（quest_handler.go）: ID と JSON を読む
//	  → usecase.AcceptQuest.Execute: 受注処理
//	  → 成功: c.JSON(200, ...) / 失敗: return err → ErrorHandler（errors.go）が JSON エラーにする
package handler
