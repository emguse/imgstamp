# バージョン管理とリリース

## 版番号の基準

`imgstamp` 自体の版番号は Git の `vMAJOR.MINOR.PATCH` タグで管理する。Go ツールチェーンや依存モジュールのバージョンとは独立している。`go.mod` に自分自身の版番号は書かない。別の `VERSION` ファイルは設けず、タグ・実行ファイル・配布物に同じ番号を使う。

初回のプレリリースは `v0.1.0-rc.1`、第2候補は `v0.1.0-rc.2`。正式版 `v0.1.0` は2026-09-29にリリースした。通常の開発ビルドの `--version` は `imgstamp dev` と表示する。

- `v0.x.y`: 初期開発。機能追加や互換性を壊す変更では minor、不具合修正では patch を上げる。破壊的変更は変更履歴に明記する。
- `v1.0.0`: CLI 引数、TOML 設定、終了コード、出力命名の互換性を維持できる段階の基準。
- v1 以降: 互換性を壊す変更は major、互換性を保つ機能追加は minor、不具合修正は patch を上げる。

変更内容は [CHANGELOG.md](../CHANGELOG.md) の `Unreleased` に追記し、リリース時に版番号と日付を付ける。公開済みタグを移動・再利用せず、修正は新しい版として出す。

## リリース手順

1. 変更履歴を更新し、対象変更をコミットして `main` に統合する。`git status --porcelain` が空であることを確認する。
2. `go test ./...`、`go test -race ./...`、`go vet ./...` を実行する。対象 OS の動作確認結果と未検証事項をリリースノートに記録する。
3. 対象コミットに注釈付きタグを作成する。例: `git tag -a v0.1.0 -m "Release v0.1.0"`。
4. `git describe --tags --exact-match HEAD` で対象タグを確認し、同じ版番号を埋め込んでビルドする。

```sh
go build -trimpath -ldflags "-X main.version=v0.1.0" -o bin/imgstamp ./cmd/imgstamp
bin/imgstamp --version
```

Windows では出力を `bin/imgstamp.exe` に変更する。上記は `v0.1.0` の例であり、対象タグに合わせて置き換える。`--version` が `imgstamp v0.1.0` になることを確認する。開発中の未コミット変更を含むバイナリにリリース番号を付けない。

5. 配布物は `imgstamp_v0.1.0_darwin_arm64`、`imgstamp_v0.1.0_windows_amd64` など版番号と対象 OS／CPU を含む名前にし、実行ファイル、`LICENSE`、`THIRD_PARTY_NOTICES.md`、設定例を同梱する。
6. 公開する際に対象タグを push し、対応する GitHub Release に配布物と変更履歴を載せる。

## Go モジュールとしての公開

モジュールパスは GitHub リポジトリに合わせて `github.com/emguse/imgstamp` とする。内部 import にも同じパスを使う。実行ファイル名は `imgstamp` のままとする。

対象コミットとタグを公開し、リポジトリへアクセスできる環境では、次の形式で導入する。以下は初回プレリリースの例。リモート導入は未検証。

```sh
go install github.com/emguse/imgstamp/cmd/imgstamp@v0.1.0
```

通常の `go install` は版番号の埋め込みを指定しないため、現行実装の `--version` は `dev` と表示する。ビルドに記録されたモジュールの版番号は `go version -m <実行ファイルのパス>` で確認する。版番号を表示する配布バイナリは、上記リリース手順の `-ldflags` 付きビルドで作成する。

版番号の基本は [Go の公式ドキュメント](https://go.dev/doc/modules/version-numbers) に従う。
