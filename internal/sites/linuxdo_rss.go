package sites

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/model"
)

type LinuxDoRSSFetcher struct {
	client *http.Client
	url    string
}

func NewLinuxDoRSSFetcher(client *http.Client, rssURL string) *LinuxDoRSSFetcher {
	if rssURL == "" {
		rssURL = "https://linux.do/latest.rss"
	}
	return &LinuxDoRSSFetcher{client: client, url: rssURL}
}

func (f *LinuxDoRSSFetcher) Site() model.SiteID { return model.SiteLinuxDO }

func (f *LinuxDoRSSFetcher) FetchLatest() ([]model.Topic, error) {
	req, _ := http.NewRequest(http.MethodGet, f.url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; forum-watch-bot/1.0)")
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9,*/*;q=0.8")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("linux.do rss http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	parser := gofeed.NewParser()
	parser.Client = f.client
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	out := make([]model.Topic, 0, len(feed.Items))
	for _, item := range feed.Items {
		publishedAt := time.Now()
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedAt = *item.UpdatedParsed
		}
		externalID := item.GUID
		if externalID == "" {
			externalID = item.Link
		}
		out = append(out, model.Topic{
			Site:        model.SiteLinuxDO,
			ExternalID:  externalID,
			CategoryID:  0,
			Category:    feed.Title,
			Title:       item.Title,
			URL:         item.Link,
			Author:      item.Author.Name,
			Excerpt:     item.Description,
			PublishedAt: publishedAt,
		})
	}
	return out, nil
}
