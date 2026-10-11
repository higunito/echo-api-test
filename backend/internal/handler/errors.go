package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
	"github.com/higunito/echo-api-test/backend/internal/usecase"
)

// errorResponse はエラー時の JSON 形式 {"error":{"code":"...","message":"..."}}（research R9）。
//
// 構造体のフィールドの後ろにある `json:"code"` は「構造体タグ」。
// encoding/json はこのタグを見て、JSON のキー名を決める（タグがなければフィールド名 Code がそのまま使われる）。
type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`    // 画面が分岐に使う機械向けのコード
	Message string `json:"message"` // 利用者にそのまま見せる日本語メッセージ（FR-012）
}

// validationError はリクエストの形式不正（クエリ・パス・JSON）を表す。
//
// Go では Error() string メソッドを持つ型はすべて error インターフェースを満たす。
// センチネルエラー（errors.New）と違い、メッセージを毎回変えられる独自のエラー型を作れる。
type validationError struct{ msg string }

func newValidationError(msg string) error { return &validationError{msg: msg} }

func (e *validationError) Error() string { return e.msg }

type errorMapping struct {
	status  int
	code    string
	message string
}

// domainErrors はドメインエラー → HTTP 応答の対応表（contracts/openapi.yaml）。
//
// `[]struct{ ... }{ {...}, {...} }` は、その場限りの無名の構造体のスライスを作る書き方。
// struct の中に errorMapping と型名だけを書くと「埋め込み」になり、
// d.status のように errorMapping のフィールドを直接参照できる。
var domainErrors = []struct {
	err error
	errorMapping
}{
	{quest.ErrInvalidTitle, errorMapping{http.StatusBadRequest, "INVALID_TITLE", "タイトルは 1〜100 文字で入力してください"}},
	{quest.ErrInvalidDescription, errorMapping{http.StatusBadRequest, "INVALID_DESCRIPTION", "説明は 2000 文字以内で入力してください"}},
	{quest.ErrInvalidReward, errorMapping{http.StatusBadRequest, "INVALID_REWARD", "報酬は 0 以上の整数で入力してください"}},
	{quest.ErrDeadlineInPast, errorMapping{http.StatusBadRequest, "DEADLINE_IN_PAST", "期限は現在より後の日時を指定してください"}},
	{quest.ErrQuestNotFound, errorMapping{http.StatusNotFound, "QUEST_NOT_FOUND", "クエストが見つかりません"}},
	{adventurer.ErrAdventurerNotFound, errorMapping{http.StatusNotFound, "ADVENTURER_NOT_FOUND", "冒険者が見つかりません"}},
	{quest.ErrQuestNotOpen, errorMapping{http.StatusConflict, "QUEST_NOT_OPEN", "このクエストは既に受注されているか、完了しています"}},
	{quest.ErrQuestExpired, errorMapping{http.StatusConflict, "QUEST_EXPIRED", "このクエストは期限切れのため受注できません"}},
	{quest.ErrQuestNotAccepted, errorMapping{http.StatusConflict, "QUEST_NOT_ACCEPTED", "このクエストはまだ受注されていません"}},
	{quest.ErrQuestAlreadyCompleted, errorMapping{http.StatusConflict, "QUEST_ALREADY_COMPLETED", "このクエストは既に完了しています"}},
	{usecase.ErrInvalidPage, errorMapping{http.StatusBadRequest, "VALIDATION_ERROR", "page は 1 以上の整数で指定してください"}},
	{quest.ErrNotAssignee, errorMapping{http.StatusForbidden, "NOT_ASSIGNEE", "完了報告できるのは受注した冒険者のみです"}},
}

// ErrorHandler は Echo の HTTPErrorHandler。すべてのエラーを {"error":{code,message}} 形式で返す。
//
// Echo ではハンドラが error を return すると、最後にこの関数が呼ばれる（router.go で e.HTTPErrorHandler に設定）。
// そのため各ハンドラは「エラーを返すだけ」でよく、HTTP ステータスの決定はここ 1 か所にまとまる。
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return // すでにレスポンスを書き始めていたら何もできない
	}
	m := mapError(err)
	if m.status >= http.StatusInternalServerError {
		// 想定外のエラー（DB 接続断など）は詳細をログにだけ出し、レスポンスには出さない（情報漏えい防止）。
		// slog は Go 標準の構造化ログ。"key", value の組で項目を渡すと JSON の項目になる。
		slog.ErrorContext(c.Request().Context(), "internal error",
			"error", err,
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
		)
	}
	if err := c.JSON(m.status, errorResponse{Error: errorBody{Code: m.code, Message: m.message}}); err != nil {
		slog.ErrorContext(c.Request().Context(), "write error response", "error", err)
	}
}

// mapError はエラーの種類から HTTP 応答を決める。上から順に調べ、どれにも当たらなければ 500。
func mapError(err error) errorMapping {
	// errors.Is: 「このエラー（または中に包まれたエラー）は、この値と同じか」を調べる。センチネルエラー向け。
	for _, d := range domainErrors {
		if errors.Is(err, d.err) {
			return d.errorMapping
		}
	}
	// errors.As: 「このエラー（または中に包まれたエラー）は、この型か」を調べ、当たればその値を ve に入れる。
	// 独自のエラー型（validationError）の中身（msg）を取り出したいときに使う。
	var ve *validationError
	if errors.As(err, &ve) {
		return errorMapping{http.StatusBadRequest, "VALIDATION_ERROR", ve.msg}
	}
	// Echo 自身が返すエラー（JSON の形式不正、存在しないパスなど）は *echo.HTTPError 型。
	var he *echo.HTTPError
	if errors.As(err, &he) {
		switch he.Code {
		case http.StatusBadRequest:
			return errorMapping{http.StatusBadRequest, "VALIDATION_ERROR", "リクエストの形式が正しくありません"}
		case http.StatusNotFound:
			return errorMapping{http.StatusNotFound, "NOT_FOUND", "リソースが見つかりません"}
		case http.StatusMethodNotAllowed:
			return errorMapping{http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"}
		}
	}
	return errorMapping{http.StatusInternalServerError, "INTERNAL_ERROR", "サーバーでエラーが発生しました"}
}
