package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/do/v2"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	"github.com/qli8racn/rss-feeder/internal/adapter/driver/htmlfetch"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	userrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/user"
	adapterrss "github.com/qli8racn/rss-feeder/internal/adapter/driver/rss"
	handlermcp "github.com/qli8racn/rss-feeder/internal/adapter/handler/mcp"
	"github.com/qli8racn/rss-feeder/internal/config"
	"github.com/qli8racn/rss-feeder/internal/domain"
	driveranthropic "github.com/qli8racn/rss-feeder/internal/driver/anthropic"
	driverfeeddiscovery "github.com/qli8racn/rss-feeder/internal/driver/feeddiscovery"
	driverhtmlfetch "github.com/qli8racn/rss-feeder/internal/driver/htmlfetch"
	driverlogger "github.com/qli8racn/rss-feeder/internal/driver/logger"
	"github.com/qli8racn/rss-feeder/internal/driver/readerdb"
	dbrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerdb/article"
	dbrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerdb/feed"
	dbrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerdb/user"
	"github.com/qli8racn/rss-feeder/internal/driver/readerpg"
	pgrepoarticle "github.com/qli8racn/rss-feeder/internal/driver/readerpg/article"
	pgrepofeed "github.com/qli8racn/rss-feeder/internal/driver/readerpg/feed"
	pgrepouser "github.com/qli8racn/rss-feeder/internal/driver/readerpg/user"
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

// isStdoutOutput は log.output が標準出力を指しているかどうかを判定する。
// "stdout" の完全一致だけでなく、/dev/stdout・/proc/self/fd/1 のような標準出力への別名パスも
// stdio transport の通信破壊につながるため併せて弾く。
func isStdoutOutput(output string) bool {
	switch output {
	case "stdout", "/dev/stdout", "/proc/self/fd/1":
		return true
	default:
		return false
	}
}

// repoRootMarker は算出したディレクトリが本当にリポジトリルートかどうかの判定に使う目印ファイル。
// config.example.yml はリポジトリに必ずコミットされているため、config.yml 未設置の初回起動でも
// 存在確認に使える。
const repoRootMarker = "internal/config/config.example.yml"

// repoRootFromExecutable は AGENTS.md 記載のビルドコマンド（`go build -o bin/mcp ./cmd/mcp`）による
// `<repo_root>/bin/mcp` という配置を前提に、実行ファイルの実パスから2階層上をリポジトリルートとして算出する。
// 算出結果が実際にリポジトリルートであることを repoRootMarker の存在で確認し、`bin/mcp` が
// 想定外の場所に配置された場合に「migration failed」等の分かりにくい後続エラーではなく、
// 原因を特定できるエラーを返す。
func repoRootFromExecutable(exePath string) (string, error) {
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("実行ファイルパス(%s)のシンボリックリンク解決に失敗しました: %w", exePath, err)
	}
	repoRoot := filepath.Dir(filepath.Dir(resolved))
	if _, err := os.Stat(filepath.Join(repoRoot, repoRootMarker)); err != nil {
		return "", fmt.Errorf(
			"実行ファイル(%s)から算出したリポジトリルート(%s)に%sが見つかりません。"+
				"bin/mcp はビルドコマンドどおり <repo_root>/bin/mcp に配置してください: %w",
			exePath, repoRoot, repoRootMarker, err,
		)
	}
	return repoRoot, nil
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
	userIDFlag := flag.String("user-id", domain.DefaultUserName, "MCPクライアントを識別するユーザーID（例: alice）。省略時は CLI/Web UI と同じ default ユーザーとして動作する")
	flag.Parse()

	// タイポによる空文字・空白のみのユーザーが users テーブルに作られてしまうのを防ぐ
	// （削除・改名機能はスコープ外のため、一度作られると手動SQLでしか消せない）。
	// なお users.name は TEXT UNIQUE（NOCASE指定なし）のため大文字小文字を区別する。
	// 「alice」と「Alice」は別ユーザー（別フィード集合）として扱われる点に注意。
	trimmedUserID := strings.TrimSpace(*userIDFlag)
	if trimmedUserID == "" {
		fmt.Fprintln(os.Stderr, "--user-id は空文字・空白のみを指定できません")
		os.Exit(1)
	}

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
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if isStdoutOutput(cfg.Log.Output) {
		fmt.Fprintln(os.Stderr, "cmd/mcp は stdio transport を使用するため、config.yml の log.output に標準出力（stdout・/dev/stdout・/proc/self/fd/1）は指定できません（stderr またはファイルパスを指定してください）")
		os.Exit(1)
	}

	if cfg.DB.IsSupabase() {
		do.Provide(i, readerpg.NewClient)
		do.Provide(i, pgrepoarticle.NewRepository)
		do.Provide(i, pgrepofeed.NewRepository)
		do.Provide(i, pgrepouser.NewRepository)
	} else {
		do.Provide(i, readerdb.NewClient)
		do.Provide(i, dbrepoarticle.NewRepository)
		do.Provide(i, dbrepofeed.NewRepository)
		do.Provide(i, dbrepouser.NewRepository)
	}
	do.Provide(i, driverrss.NewReader)
	do.Provide(i, driverhtmlfetch.NewFetcher)
	do.Provide(i, driveranthropic.NewEnrichAgent)
	do.Provide(i, driveranthropic.NewPreferenceAgent)

	db, err := do.Invoke[*sql.DB](i)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := migration.RunFor(cfg, db); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	// --user-id で指定された識別子のユーザーを解決する（初回はレコードを作成するupsert方式）。
	// preferenceAgent は *domain.User をDIコンテナ経由で取得する
	// （internal/driver/anthropic/preference.go 参照）。
	resolveUserUC := usecase.NewResolveUserUsecase(do.MustInvoke[userrepo.Repository](i))
	user, err := resolveUserUC.Execute(context.Background(), trimmedUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "user resolution failed: %v\n", err)
		os.Exit(1)
	}
	do.ProvideValue(i, user)
	userID := user.ID

	logger := do.MustInvoke[*slog.Logger](i)

	// フィードURL自動探索のAIフォールバックは bin/rss-agent をサブプロセスとして呼び出す
	// 既存実装をそのまま使う（cmd/web・cmd/rss-feeder と同じ方式）。MCPサーバーは単一の
	// Claude Desktopセッションからの逐次呼び出しを想定するため、同時実行数は1に抑える。
	feedDiscoveryAgent := driverfeeddiscovery.NewSubprocessAgent(*rssAgentPath, 1)

	listUC := usecase.NewListUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	searchUC := usecase.NewSearchUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	categoriesUC := usecase.NewListCategoriesUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	listFeedsUC := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	bookmarkUC := usecase.NewBookmarkUsecase(do.MustInvoke[articlerepo.Repository](i), userID)
	addFeedUC := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	resolveFeedURLUC := usecase.NewResolveFeedURLUsecase(
		do.MustInvoke[adapterrss.RSSReader](i),
		do.MustInvoke[htmlfetch.Fetcher](i),
		feedDiscoveryAgent,
	)
	fetchUC := usecase.NewFetchUsecase(
		do.MustInvoke[articlerepo.Repository](i),
		do.MustInvoke[feedrepo.Repository](i),
		do.MustInvoke[adapterrss.RSSReader](i),
		userID,
	)
	removeFeedUC := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i), userID)
	enrichUC := usecase.NewEnrichUsecase(do.MustInvoke[adapteranthropic.EnrichAgent](i))
	preferenceUC := usecase.NewPreferenceUsecase(do.MustInvoke[adapteranthropic.PreferenceAgent](i))
	markReadUC := usecase.NewMarkReadUsecase(do.MustInvoke[articlerepo.Repository](i), userID)

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: serverName, Version: serverVersion}, nil)

	// ツール名には rss_ 接頭辞を付ける。Claude Desktop 等に複数の MCP サーバーを登録した環境では
	// "list"・"search" のような汎用的な名前だとLLMがどのドメインのツールか判別しづらいため。
	// description も「RSSフィーダーに保存済みの記事を〜」のように主語を明示する。
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_list",
		Description: "RSSフィーダーに保存済みの記事を一覧表示する。デフォルトは未読のみ。all・bookmarked・category で絞り込み可能（all と bookmarked は同時指定不可）。limit・page でページネーションし、応答の total で絞り込み条件に一致する総数を返す（デフォルト50件・上限200件）。閲覧による既読化は行わない読み取り専用のツール（既読にしたい場合は rss_mark_read を使う）。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.ListTool(listUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_search",
		Description: "RSSフィーダーに保存済みの記事をキーワードで全文検索する。bookmarked・category で絞り込み可能。limit・page でページネーションし、応答の total で検索条件に一致する総数を返す（デフォルト50件・上限200件）。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.SearchTool(searchUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_categories",
		Description: "RSSフィーダーに保存済みの記事に付与済みのカテゴリ一覧を表示する。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.CategoriesTool(categoriesUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_list_feeds",
		Description: "RSSフィーダーに登録済みのRSSフィード一覧を表示する。",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, handlermcp.ListFeedsTool(listFeedsUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_bookmark",
		Description: "RSSフィーダーで指定した記事IDのブックマーク登録/解除をトグルする。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.BookmarkTool(bookmarkUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_mark_read",
		Description: "RSSフィーダーで指定した記事IDを既読にする。rss_list は閲覧では既読化しないため、既読管理をしたい場合はこのツールを使う。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.MarkReadTool(markReadUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "rss_add_feed",
		Description: "RSS/AtomフィードのURL、またはそのフィードを持つサイトのURLをRSSフィーダーのDBに登録し、" +
			"登録直後に記事を1回取得する。フィードURLの探索を伴うため実行に数秒〜数十秒かかることがある。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.AddFeedTool(addFeedUC, resolveFeedURLUC, fetchUC, logger))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "rss_fetch",
		Description: "RSSフィーダーに登録済みの全フィードを取得してDBに保存する。ネットワークI/Oのため数秒〜数十秒かかることがある。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.FetchTool(fetchUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "rss_remove_feed",
		Description: "RSSフィーダーから指定したフィードと、それに紐づく記事を完全に削除する破壊的操作(元に戻せない)。" +
			"実行前に必ずユーザーに削除対象(フィード名・URL。rss_list_feeds で確認できる)を提示し、" +
			"明示的な同意を得てから confirm:true を渡すこと。ユーザーの同意なしに true を渡してはならない。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, handlermcp.RemoveFeedTool(removeFeedUC, listFeedsUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "rss_enrich",
		Description: "RSSフィーダーに保存済みの記事にAIによる要約・カテゴリを付与してDBに保存する。ANTHROPIC_API_KEYによる追加課金が" +
			"発生するため、実行前に必ずユーザーに『ANTHROPIC_API_KEYを使用した追加料金が発生しますが、実行して" +
			"よいですか？』と確認し、明示的な同意を得てから confirm:true を渡すこと。ユーザーの同意なしに true " +
			"を渡してはならない。limit で処理件数の上限を必ず指定すること(無制限実行を避けるため。サーバー側でも" +
			"limit・batch_size は100、concurrency は5を上限にクランプされる)。",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: boolPtr(false)},
	}, handlermcp.EnrichTool(enrichUC))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "rss_preference",
		Description: "RSSフィーダーでブックマーク済みの記事からユーザーの趣向を分析する(DB更新は行わない読み取り専用処理)。" +
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
