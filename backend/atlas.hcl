// Atlas の設定。スキーマの正は db/schema.sql（research R4）。

locals {
  src     = "file://db/schema.sql"
  dev     = "docker://postgres/18/dev?search_path=public"
  mig_dir = "file://db/migrations"
}

env "local" {
  src = local.src
  url = getenv("DATABASE_URL")
  dev = local.dev
  migration {
    dir = local.mig_dir
  }
}

env "test" {
  src = local.src
  url = getenv("TEST_DATABASE_URL")
  dev = local.dev
  migration {
    dir = local.mig_dir
  }
}
