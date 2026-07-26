package mcp

import "testing"

func TestResolvePerPage(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		want  int
	}{
		{"未指定(0)はデフォルト", 0, defaultPerPage},
		{"負値はデフォルト", -1, defaultPerPage},
		{"上限内はそのまま", 100, 100},
		{"上限ちょうど", maxPerPage, maxPerPage},
		{"上限超過はクランプ", maxPerPage + 1, maxPerPage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolvePerPage(c.limit); got != c.want {
				t.Errorf("resolvePerPage(%d): got %d, want %d", c.limit, got, c.want)
			}
		})
	}
}

func TestResolvePage(t *testing.T) {
	cases := []struct {
		name string
		page int
		want int
	}{
		{"未指定(0)は1ページ目", 0, 1},
		{"負値は1ページ目", -5, 1},
		{"指定値はそのまま", 3, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolvePage(c.page); got != c.want {
				t.Errorf("resolvePage(%d): got %d, want %d", c.page, got, c.want)
			}
		})
	}
}
