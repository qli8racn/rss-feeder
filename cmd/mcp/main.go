package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/samber/do/v2"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	handlermcp "github.com/qli8racn/rss-feeder/internal/adapter/handler/mcp"
	"github.com/qli8racn/rss-feeder/internal/config"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	driverlogger "github.com/qli8racn/rss-feeder/internal/driver/logger"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	driverrss "github.com/qli8racn/rss-feeder/internal/driver/rss"
	"github.com/qli8racn/rss-feeder/internal/migration"
	"github.com/qli8racn/rss-feeder/internal/usecase"
)

// serverName・serverVersion は Claude Desktop 等の MCP クライアントに表示されるサーバー識別情報。
const (
	serverName    = "rss-feeder"
	serverVersion = "0.1.0"
)

func boolPtr(b bool) *bool { return &b }

// repoRootFromExecutable は AGENTS.md 記載のビルドコマンド（`go build -o bin/mcp ./cmd/mcp`）による
// `<repo_root>/bin/mcp` という配置を前提に、実行ファイルの実パスから2階層上をリポジトリルートとして算出する。
func repoRootFromExecutable(exePath string) (string, error) {
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("実行ファイルパス(%s)のシンボリックリンク解決に失敗しました: %w", exePath, err)
	}
	return filepath.Dir(filepath.Dir(resolved)), nil
}

// chdirToRepoRoot は cmd/mcp をリポジトリルートで実行しているかのように振る舞わせるための処理。
// Claude Desktop 等の MCP クライアントはサブプロセスをリポジトリの作業ディレクトリに cd してから
// 起動してくれないため、internal/driver/readerdb・internal/config が前提とする「CWD == リポジトリ
// ルート」の相対パスをそのまま使い続けられるよう、DI 配線より前にプロセスの CWD を変更する
// （cmd/web・cmd/rss-feeder・cmd/agent は従来どおりリポジトリルートから起動される運用のため対象外）。
func chdirToRepoRoot() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("実行ファイルパスの取得に失敗しました: %w", err)
	}
	repoRoot, err := repoRootFromExecutable(exe)
	if err != nil {
		return err
	}
	if err := os.Chdir(repoRoot); err != nil {
		return fmt.Errorf("リポジトリルート(%s)へのディレクトリ変更に失敗しました: %w", repoRoot, err)
	}
	return nil
}

func main() {
	rssAgentPath := flag.String("rss-agent-path", "bin/rss-agent", "フィードURL自動探索のAIフォールバックに使う rss-agent バイナリのパス")
	flag.Parse()

	// DB・config.yml の相対パス解決より前に、必ずリポジトリルートへ cd する。
	if err := chdirToRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// enrich・preference ツールが Anthropic API を利用するため、cmd/agent と同様に起動時に
	// ANTHROPIC_API_KEY を環境変数へ設定する。
	if err := config.SetupAnthropicAPIKey(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	i := do.New()
	do.Provide(i, config.NewProvider)
	do.Provide(i, driverlogger.NewLogger)

	// stdio transport では標準出力(stdout)がJSON-RPC通信そのものに使われるため、
	// ログ出力先が stdout だと通信を破壊してしまう。誤設定に気づけるよう、DI配線を
	// 進める前に検知して起動を中断する（docs/steering/20260726_mcp_server/design.md 参照）。
	cfg := do.MustInvoke[*config.Config](i)
	if cfg.Log.Output == "stdout" {
		fmt.Fprintln(os.Stderr, "cmd/mcp は stdio transport を使用するため、config.yml の log.output に stdout は指定できません（stderr またはファイルパスを指定してください）")
		os.Exit(1)
	}

	do.Provide(i, readerdb.NewClient)
	do.Provide(i, dbrepoarticle.NewRepository)
	do.Provide(i, dbrepofeed.NewRepository)
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)
	do.Provide(i, driveranthropic.NewEnrichAgent)
	do.Provide(i, driveranthropic.NewPreferenceAgent)

	db := do.MustInvoke[*sql.DB](i)
	if err := migration.Run(db); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	logger := do.MustInvoke[*slog.Logger](i)

	// フィードURL自動探索のAIフォールバックは bin/rss-agent をサブプロセスとして呼び出す
	// 既存実装をそのまま使う（cmd/web・cmd/rss-feeder と同じ方式）。MCPサーバーは単一の
	// Claude Desktopセッションからの逐次呼び出しを想定するため、同時実行数は1に抑える。
	feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, 1)

	listUC := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i))
	searchUC := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i))
	categoriesUC := usecase.NewListCategoriesUsecase(do.MustInvoke[articlerepo.Repository](i))
	listFeedsUC := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i))
	bookmarkUC := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i))
	addFeedUC := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
	resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
		do.MustInvoke[adapterrss.RSSReader](i),
		do.MustInvoke[htmlfetch.Fetcher](i),
		feedDiscoveryAgent,
	)
	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
	)
	removeFeedUC := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
	enrichUC := usecase.NewEnrichUsecase(do.MustInvoke[adapteranthropic.EnrichAgent](i))
	preferenceUC := usecase.NewPreferenceUsecase(do.MustInvoke[adapteranthropic.PreferenceAgent](i))

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: serverName, Version: serverVersion}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list",
		Description: "保存済み記事を一覧表示する。デフォルトは未読のみ。all・bookmarked・category で絞り込み可能（all と bookmarked は同時指定不可）。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.ListTool(listUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "search",
		Description: "キーワードで記事を全文検索する。bookmarked・category で絞り込み可能。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.SearchTool(searchUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "categories",
		Description: "記事に付与済みのカテゴリ一覧を表示する。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.CategoriesTool(categoriesUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list-feeds",
		Description: "登録済みRSSフィード一覧を表示する。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.ListFeedsTool(listFeedsUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "bookmark",
		Description: "指定した記事IDのブックマーク登録/解除をトグルする。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.BookmarkTool(bookmarkUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "add-feed",
		Description: "RSS/AtomフィードのURL、またはそのフィードを持つサイトのURLをDBに登録し、" +
			"登録直後に記事を1回取得する。フィードURLの探索を伴うため実行に数秒〜数十秒かかることがある。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.AddFeedTool(addFeedUC, resolveFeedURLUC, fetchUC, logger))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "fetch",
		Description: "登録済みの全フィードを取得してDBに保存する。ネットワークI/Oのため数秒〜数十秒かかることがある。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.FetchTool(fetchUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "remove-feed",
		Description: "指定したフィードと、それに紐づく記事を完全に削除する破壊的操作(元に戻せない)。" +
			"実行前に必ずユーザーに削除対象(フィード名・記事件数など。list-feeds・list で確認できる)を提示し、" +
			"明示的な同意を得てから confirm:true を渡すこと。ユーザーの同意なしに true を渡してはならない。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, handlermcp.RemoveFeedTool(removeFeedUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "enrich",
		Description: "記事にAIによる要約・カテゴリを付与してDBに保存する。ANTHROPIC_API_KEYによる追加課金が" +
			"発生するため、実行前に必ずユーザーに『ANTHROPIC_API_KEYを使用した追加料金が発生しますが、実行して" +
			"よいですか？』と確認し、明示的な同意を得てから confirm:true を渡すこと。ユーザーの同意なしに true " +
			"を渡してはならない。limit で処理件数の上限を必ず指定すること(無制限実行を避けるため)。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.EnrichTool(enrichUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "preference",
		Description: "ブックマーク済み記事からユーザーの趣向を分析する(DB更新は行わない読み取り専用処理)。" +
			"ANTHROPIC_API_KEYによる追加課金が発生するため、実行前に必ずユーザーに『ANTHROPIC_API_KEYを使用した" +
			"追加料金が発生しますが、実行してよいですか？』と確認し、明示的な同意を得てから confirm:true を渡す" +
			"こと。ユーザーの同意なしに true を渡してはならない。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.PreferenceTool(preferenceUC))

	if err := server.Run(context.Background(), &mcpsdk.StdioTransport{}); err != nil {
		logger.Error("mcpサーバーが停止しました", "error", err)
		os.Exit(1)
	}
}
