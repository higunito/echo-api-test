// Command server は API サーバのエントリポイント。
//
// Go では package main の main 関数がプログラムの開始地点になる。
// `go run ./cmd/server`（mise run dev:backend）で起動する。
// cmd/ 配下に実行可能なプログラムを置き、ロジックは internal/ に置くのが Go のよくある構成。
// （internal/ 配下のパッケージは、このモジュールの外からは import できない）
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/higunito/echo-api-test/backend/internal/handler"
	"github.com/higunito/echo-api-test/backend/internal/infrastructure/postgres"
	"github.com/higunito/echo-api-test/backend/internal/usecase"
)

const defaultDatabaseURL = "postgres://guild:guild@localhost:5432/guild?sslmode=disable"

func main() {
	// ログを JSON 形式で標準出力に出す。以後 slog.Info(...) などはこの設定で出力される。
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	// main 自体は error を返せないので、処理は run() に分けて、エラーなら終了コード 1 で終わる。
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Ctrl+C（SIGINT）や SIGTERM を受けると ctx がキャンセルされる。
	// defer は「この関数を抜けるときに実行する」予約。後片付けを書き忘れないための Go の定番。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, getenv("DATABASE_URL", defaultDatabaseURL))
	if err != nil {
		return err
	}
	defer pool.Close()

	// ---- 依存の組み立て（手動 DI）----
	// 内側から順に作って外側に渡す: リポジトリ（infrastructure）→ usecase → handler → ルータ。
	// usecase は domain のインターフェースしか知らないので、ここで具体的な Postgres 実装を渡す。
	adventurerRepo := postgres.NewAdventurerRepository(pool)
	questRepo := postgres.NewQuestRepository(pool)
	clock := usecase.SystemClock{}

	e := handler.NewRouter(
		handler.NewAdventurerHandler(usecase.NewListAdventurers(adventurerRepo)),
		handler.NewQuestHandler(handler.QuestHandlerDeps{
			Create:   usecase.NewCreateQuest(questRepo, clock),
			List:     usecase.NewListQuests(questRepo, adventurerRepo, clock),
			Get:      usecase.NewGetQuest(questRepo, adventurerRepo),
			Accept:   usecase.NewAcceptQuest(questRepo, adventurerRepo, clock),
			Complete: usecase.NewCompleteQuest(questRepo, adventurerRepo, clock),
			Clock:    clock,
		}),
	)

	// ---- 起動と終了（グレースフルシャットダウン）----
	addr := ":" + getenv("PORT", "8080")
	// チャネル（chan）は goroutine 間で値を受け渡す仕組み。バッファ 1 なので送信側は待たずに済む。
	errCh := make(chan error, 1)
	// go func() { ... }() で関数を goroutine（軽量スレッド）として並行に動かす。
	// e.Start はサーバが止まるまで戻らないので、別 goroutine で動かし、main 側はシグナルを待つ。
	go func() {
		slog.Info("server started", "addr", addr)
		// Shutdown で止めたときは http.ErrServerClosed が返る（正常終了なのでエラー扱いしない）
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // チャネルにエラーを送る
		}
		close(errCh)
	}()

	// select は複数のチャネルのうち、先に受信できたものの case を実行する。
	select {
	case err := <-errCh: // サーバが起動に失敗した（ポート使用中など）
		return err
	case <-ctx.Done(): // Ctrl+C などで終了を指示された
	}
	// 処理中のリクエストが終わるのを最大 10 秒待ってから止める。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return e.Shutdown(shutdownCtx)
}

// getenv は環境変数を読み、未設定なら fallback を返す。
// mise.toml の [env] で DATABASE_URL などを設定している。
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
