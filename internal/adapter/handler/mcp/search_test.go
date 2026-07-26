package mcp

import (
	"context"
	"errors"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchTool_EmptyKeywordRejected(t *testing.T) {
	// keyword の空文字チェックは usecase を呼び出す前に弾かれるため、
	// SearchUsecase は nil のままで検証できる。
	tool := SearchTool(nil)

	_, _, err := tool(context.Background(), &mcpsdk.CallToolRequest{}, SearchInput{Keyword: ""})
	if !errors.Is(err, ErrSearchKeywordRequired) {
		t.Errorf("err: got %v, want ErrSearchKeywordRequired", err)
	}
}
