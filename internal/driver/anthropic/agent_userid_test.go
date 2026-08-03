package anthropic

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	adapteranthropic "github.com/qli8racn/rss-feeder/internal/adapter/driver/anthropic"
	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	feedrepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/feed"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

// testAgentUserID は preference/curate/discover/summarize の各Agentが、DIコンテナから取得した
// userIDをrepo呼び出しに正しく伝播していることを検証するための固定値。
const testAgentUserID int64 = 99

// fakeToolUseMessage は Claude が単一のツール呼び出しを要求したレスポンスを組み立てる。
// fakeMessage（internal/driver/anthropic/enrich_test.go）と同様、JSON経由でデコードする
// 必要があるため（ContentBlockUnion.AsAny() はデコード時に保持される生バイト列を参照する）。
func fakeToolUseMessage(t *testing.T, toolName string) *anthropic.Message {
	t.Helper()
	payload := map[string]any{
		"id":   "msg_test",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{"type": "tool_use", "id": "toolu_1", "name": toolName, "input": map[string]any{}},
		},
		"model":       "claude-opus-4-8",
		"stop_reason": anthropic.StopReasonToolUse,
		"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("fakeToolUseMessage: marshal: %v", err)
	}
	var msg anthropic.Message
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatalf("fakeToolUseMessage: unmarshal: %v", err)
	}
	return &msg
}

// fakeFinalTextMessage はツール呼び出し後の最終応答（テキストのみ）を組み立てる。
func fakeFinalTextMessage(t *testing.T, text string) *anthropic.Message {
	t.Helper()
	return fakeMessage(t, text, anthropic.Usage{InputTokens: 5, OutputTokens: 5}, anthropic.StopReasonEndTurn)
}

// toolUseThenFinal は1回目の呼び出しでtoolNameのツール呼び出しを要求し、
// 2回目以降は最終テキスト応答を返すfakeクライアントを組み立てる。
func toolUseThenFinal(t *testing.T, toolName, finalText string) func(context.Context, anthropic.MessageNewParams, ...option.RequestOption) (*anthropic.Message, error) {
	calls := 0
	return func(context.Context, anthropic.MessageNewParams, ...option.RequestOption) (*anthropic.Message, error) {
		calls++
		if calls == 1 {
			return fakeToolUseMessage(t, toolName), nil
		}
		return fakeFinalTextMessage(t, finalText), nil
	}
}

// fakeArticleRepoForUserID は articlerepo.Repository を埋め込み、FindBookmarked と ListAll
// （internal/driver/readerdb/feed の Repository と合わせてテストする discoverAgent 用）だけを
// 上書きする（enrich_test.go の fakeRepo と同じパターン）。
type fakeArticleRepoForUserID struct {
	articlerepo.Repository
	findBookmarked func(ctx context.Context, userID int64) ([]domain.Article, error)
	fetchLatest    func(ctx context.Context, limit int, feedURL string, userID int64) ([]domain.Article, error)
}

func (f *fakeArticleRepoForUserID) FindBookmarked(ctx context.Context, userID int64) ([]domain.Article, error) {
	return f.findBookmarked(ctx, userID)
}

func (f *fakeArticleRepoForUserID) FetchLatest(ctx context.Context, limit int, feedURL string, userID int64) ([]domain.Article, error) {
	return f.fetchLatest(ctx, limit, feedURL, userID)
}

type fakeFeedRepoForUserID struct {
	feedrepo.Repository
	listAll func(ctx context.Context, userID int64) ([]domain.Feed, error)
}

func (f *fakeFeedRepoForUserID) ListAll(ctx context.Context, userID int64) ([]domain.Feed, error) {
	return f.listAll(ctx, userID)
}

func TestPreferenceAgent_Run_PassesUserIDToFindBookmarked(t *testing.T) {
	var gotUserID int64
	repo := &fakeArticleRepoForUserID{
		findBookmarked: func(_ context.Context, userID int64) ([]domain.Article, error) {
			gotUserID = userID
			return []domain.Article{{ID: 1, Title: "A"}}, nil
		},
	}
	agent := &preferenceAgent{
		logger: discardLogger(),
		client: &fakeMessageCreator{new: toolUseThenFinal(t, "fetch_bookmarked", "分析結果")},
		reader: repo,
		userID: testAgentUserID,
	}

	if _, err := agent.Run(context.Background()); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if gotUserID != testAgentUserID {
		t.Errorf("FindBookmarked userID: got %d, want %d", gotUserID, testAgentUserID)
	}
}

func TestCurateAgent_Run_PassesUserIDToFindBookmarked(t *testing.T) {
	var gotUserID int64
	repo := &fakeArticleRepoForUserID{
		findBookmarked: func(_ context.Context, userID int64) ([]domain.Article, error) {
			gotUserID = userID
			return []domain.Article{{ID: 1, Title: "A"}}, nil
		},
	}
	agent := &curateAgent{
		logger: discardLogger(),
		client: &fakeMessageCreator{new: toolUseThenFinal(t, "fetch_bookmarked_articles", "推薦結果")},
		repo:   repo,
		userID: testAgentUserID,
	}

	if _, err := agent.Run(context.Background(), adapteranthropic.CurateOptions{}); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if gotUserID != testAgentUserID {
		t.Errorf("FindBookmarked userID: got %d, want %d", gotUserID, testAgentUserID)
	}
}

func TestDiscoverAgent_Run_PassesUserIDToListAll(t *testing.T) {
	var gotUserID int64
	feeds := &fakeFeedRepoForUserID{
		listAll: func(_ context.Context, userID int64) ([]domain.Feed, error) {
			gotUserID = userID
			return []domain.Feed{{ID: 1, FeedURL: "https://example.com/feed"}}, nil
		},
	}
	agent := &discoverAgent{
		logger:      discardLogger(),
		client:      &fakeMessageCreator{new: toolUseThenFinal(t, "fetch_registered_feeds", "推薦フィード")},
		articleRepo: &fakeArticleRepoForUserID{findBookmarked: func(_ context.Context, _ int64) ([]domain.Article, error) { return nil, nil }},
		feedRepo:    feeds,
		userID:      testAgentUserID,
	}

	if _, err := agent.Run(context.Background()); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if gotUserID != testAgentUserID {
		t.Errorf("ListAll userID: got %d, want %d", gotUserID, testAgentUserID)
	}
}

func TestSummarizeAgent_Run_PassesUserIDToFetchLatest(t *testing.T) {
	var gotUserID int64
	repo := &fakeArticleRepoForUserID{
		fetchLatest: func(_ context.Context, _ int, _ string, userID int64) ([]domain.Article, error) {
			gotUserID = userID
			return []domain.Article{{ID: 1, Title: "A"}}, nil
		},
	}
	agent := &summarizeAgent{
		logger: discardLogger(),
		client: &fakeMessageCreator{new: toolUseThenFinal(t, "fetch_articles", "要約結果")},
		reader: repo,
		userID: testAgentUserID,
	}

	if _, err := agent.Run(context.Background(), adapteranthropic.SummarizeOptions{}); err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if gotUserID != testAgentUserID {
		t.Errorf("FetchLatest userID: got %d, want %d", gotUserID, testAgentUserID)
	}
}
