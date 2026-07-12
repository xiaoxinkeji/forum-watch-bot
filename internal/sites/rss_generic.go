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

type GenericRSSFetcher struct {
	client *http.Client
	site   model.SiteID
	url    string
}

func NewGenericRSSFetcher(client *http.Client, site model.SiteID, rssURL string) *GenericRSSFetcher {
	return &GenericRSSFetcher{client: client, site: site, url: rssURL}
}

func (f *GenericRSSFetcher) Site() model.SiteID { return f.site }

func (f *GenericRSSFetcher) FetchLatest() ([]model.Topic, error) {
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
		return nil, fmt.Errorf("%s rss http %d: %s", f.site, resp.StatusCode, strings.TrimSpace(string(body)))
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
		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}
		out = append(out, model.Topic{
			Site:        f.site,
			ExternalID:  externalID,
			CategoryID:  0,
			Category:    feed.Title,
			Title:       item.Title,
			URL:         item.Link,
			Author:      author,
			Excerpt:     item.Description,
			PublishedAt: publishedAt,
		})
	}
	return out, nil
}
