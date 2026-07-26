package mcp

import "errors"

// ErrConfirmationRequired は破壊的操作（remove-feed）・課金が発生する操作（enrich・preference）で
// confirm=true が明示的に渡されなかった場合に返す。MCPクライアント（LLM）に対し、ユーザーへの
// 事前確認なしに confirm:true を渡してはならない旨をツールの description で指示しているが、
// サーバー側でも同意なしの実行を防ぐための最終防御として設ける。
var ErrConfirmationRequired = errors.New("confirm=true による明示的な同意なしにこの操作は実行できません。事前にユーザーへ確認し、同意を得てから再実行してください")

func requireConfirm(confirm bool) error {
	if !confirm {
		return ErrConfirmationRequired
	}
	return nil
}
