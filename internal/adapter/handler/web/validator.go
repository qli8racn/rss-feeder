package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// NewRequestValidatorMiddleware は OpenAPI 仕様（specPath）に基づき、
// クエリ・パスパラメータ・リクエストボディを検証する chi 互換の net/http ミドルウェアを構築する。
// 仕様に存在しないパスへのリクエストは 404 として弾かれるため、`/api` 配下のルーターにのみ適用すること。
func NewRequestValidatorMiddleware(specPath string) (func(http.Handler) http.Handler, error) {
	doc, err := openapi3.NewLoader().LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}

	return nethttpmiddleware.OapiRequestValidatorWithOptions(doc, &nethttpmiddleware.Options{
		// servers: ["/"] の相対URLによる Host 検証の誤判定を避ける（仕様は単一サーバー運用のため不要）。
		DoNotValidateServers: true,
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			writeJSONError(w, statusCode, message)
		},
	}), nil
}
