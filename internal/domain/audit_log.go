package domain

type Action string

const (
	ActionFetch    Action = "fetch"
	ActionBookmark Action = "bookmark"
	ActionReset    Action = "reset"
)

type AuditLog struct {
	Action    Action
	ArticleID *int64
	OldState  string
	NewState  string
}
