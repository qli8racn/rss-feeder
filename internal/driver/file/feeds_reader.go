package file

import (
	"bufio"
	"os"
	"strings"

	adapterfile "github.com/qli8racn/rss-feeder/internal/adapter/driver/file"
	"github.com/samber/do/v2"
)

type feedsReader struct {
	path string
}

func NewFeedsReader(_ do.Injector) (adapterfile.FeedsReader, error) {
	return &feedsReader{path: "feeds.txt"}, nil
}

func (r *feedsReader) Load() ([]string, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, scanner.Err()
}
