package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenWriter_Stderr(t *testing.T) {
	for _, output := range []string{"", "stderr"} {
		t.Run(output, func(t *testing.T) {
			w, closer, err := openWriter(output)
			if err != nil {
				t.Fatalf("openWriter(%q): unexpected error: %v", output, err)
			}
			if w != os.Stderr {
				t.Errorf("openWriter(%q): got %v, want os.Stderr", output, w)
			}
			if closer != nil {
				t.Errorf("openWriter(%q): closer should be nil for stderr", output)
			}
		})
	}
}

func TestOpenWriter_Stdout(t *testing.T) {
	w, closer, err := openWriter("stdout")
	if err != nil {
		t.Fatalf("openWriter(stdout): unexpected error: %v", err)
	}
	if w != os.Stdout {
		t.Errorf("openWriter(stdout): got %v, want os.Stdout", w)
	}
	if closer != nil {
		t.Errorf("openWriter(stdout): closer should be nil for stdout")
	}
}

func TestOpenWriter_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test_logger.log")

	w, closer, err := openWriter(path)
	if err != nil {
		t.Fatalf("openWriter(file): unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("openWriter(file): writer should not be nil")
	}
	if closer == nil {
		t.Fatal("openWriter(file): closer should not be nil for file output")
	}
	if err := closer.Close(); err != nil {
		t.Errorf("closer.Close(): unexpected error: %v", err)
	}
}

func TestOpenWriter_NonexistentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "sub", "test.log")

	_, _, err := openWriter(path)
	if err == nil {
		t.Fatal("openWriter(nonexistent dir): expected error, got nil")
	}
}

func TestNewHandler_JSON(t *testing.T) {
	h := newHandler(os.Stderr, "json")
	if _, ok := h.(*slog.JSONHandler); !ok {
		t.Errorf("newHandler(json): got %T, want *slog.JSONHandler", h)
	}
}

func TestNewHandler_Text(t *testing.T) {
	h := newHandler(os.Stderr, "text")
	if _, ok := h.(*slog.TextHandler); !ok {
		t.Errorf("newHandler(text): got %T, want *slog.TextHandler", h)
	}
}

func TestNewHandler_Default(t *testing.T) {
	h := newHandler(os.Stderr, "")
	if _, ok := h.(*slog.TextHandler); !ok {
		t.Errorf("newHandler(empty): got %T, want *slog.TextHandler (default)", h)
	}
}
