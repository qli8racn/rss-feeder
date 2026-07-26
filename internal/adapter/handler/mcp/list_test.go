package mcp

import (
	"context"
	"errors"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListTool_AllAndBookmarkedExclusive(t *testing.T) {
	// all・bookmarked の同時指定は usecase を呼び出す前に弾かれるため、
	// ListUsecase は nil のままで検証できる。
	tool := ListTool(nil)

	_, _, err := tool(context.Background(), &mcpsdk.CallToolRequest{}, ListInput{All: true, Bookmarked: true})
	if !errors.Is(err, ErrListAllAndBookmarkedExclusive) {
		t.Errorf("err: got %v, want ErrListAllAndBookmarkedExclusive", err)
	}
}
