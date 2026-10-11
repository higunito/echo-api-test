package handler

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewRouter はミドルウェアとルーティングを設定した Echo を返す。
// main.go から呼ばれ、テスト（quest_handler_test.go の serve）でも同じルータを使う。
func NewRouter(adventurers *AdventurerHandler, quests *QuestHandler) *echo.Echo {
	e := echo.New()
	e.HideBanner = true               // 起動時の ASCII アートを出さない
	e.HTTPErrorHandler = ErrorHandler // ハンドラが返した error をここで JSON にする（errors.go）

	// ミドルウェアは登録した順に、すべてのリクエストの前後で実行される。
	e.Use(middleware.RequestID()) // X-Request-Id ヘッダを付け、ログとレスポンスを結び付けられるようにする
	e.Use(middleware.Recover())   // ハンドラ内で panic が起きてもサーバを落とさず 500 を返す
	// アクセスログ。LogXxx: true にした項目が v（RequestLoggerValues）に入り、LogValuesFunc で出力する。
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		// HandleError: true にすると、エラー時も ErrorHandler が決めたステータスがログに出る
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.InfoContext(c.Request().Context(), "request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency_ms", v.Latency.Milliseconds(),
				"request_id", v.RequestID,
			)
			return nil
		},
	}))

	// Group で共通のパス接頭辞 /api をまとめる。ここに登録したルートはすべて /api/... になる。
	api := e.Group("/api")
	// 第 2 引数はメソッド値（adventurers.List）。レシーバを束縛した関数として渡せる。
	api.GET("/adventurers", adventurers.List)
	quests.register(api)
	return e
}
