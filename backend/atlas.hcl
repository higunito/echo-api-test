// Atlas の設定（HCL 形式）。スキーマの正は db/schema.sql（research R4）。
//
// env ブロックごとに接続先を切り替える。`atlas migrate apply --env local` のように --env で選ぶ。
//   src : あるべきスキーマ（schema.sql）
//   url : マイグレーションを適用する DB（mise.toml の [env] で設定した環境変数から読む）
//   dev : 差分計算用の使い捨て DB。Atlas が Docker で一時的に Postgres を起動し、
//         schema.sql とマイグレーション履歴をそれぞれ流して比較する
//   migration.dir : マイグレーションファイルと atlas.sum の置き場所

locals {
  src     = "file://db/schema.sql"
  dev     = "docker://postgres/18/dev?search_path=public"
  mig_dir = "file://db/migrations"
}

// 開発用 DB（compose.yaml の db サービス, :5432）
env "local" {
  src = local.src
  url = getenv("DATABASE_URL")
  dev = local.dev
  migration {
    dir = local.mig_dir
  }
}

// 統合テスト用 DB（compose.yaml の db-test サービス, :5433）
env "test" {
  src = local.src
  url = getenv("TEST_DATABASE_URL")
  dev = local.dev
  migration {
    dir = local.mig_dir
  }
}
