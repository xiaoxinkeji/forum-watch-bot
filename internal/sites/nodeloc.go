package sites

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoxinkeji/forum-watch-bot/internal/model"
)

type NodeLocFetcher struct {
	client  *http.Client
	baseURL string
}

func NewNodeLocFetcher(client *http.Client) *NodeLocFetcher {
	return &NodeLocFetcher{client: client, baseURL: "https://www.nodeloc.com"}
}

func (f *NodeLocFetcher) Site() model.SiteID { return model.SiteNodeLoc }

type discourseLatestResp struct {
	TopicList struct {
		Topics []struct {
			ID              int    `json:"id"`
			Title           string `json:"title"`
			Slug            string `json:"slug"`
			CategoryID      int    `json:"category_id"`
			PostsCount      int    `json:"posts_count"`
			CreatedAt       string `json:"created_at"`
			LastPostedAt    string `json:"last_posted_at"`
			Excerpt         string `json:"excerpt"`
			FancyTitle      string `json:"fancy_title"`
			HighestPostNumber int  `json:"highest_post_number"`
		} `json:"topics"`
	} `json:"topic_list"`
}

func (f *NodeLocFetcher) FetchLatest() ([]model.Topic, error) {
	req, _ := http.NewRequest(http.MethodGet, f.baseURL+"/latest.json", nil)
	req.Header.Set("User-Agent", "forum-watch-bot/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("nodeloc latest.json http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed discourseLatestResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	out := make([]model.Topic, 0, len(parsed.TopicList.Topics))
	for _, t := range parsed.TopicList.Topics {
		publishedAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
		if publishedAt.IsZero() {
			publishedAt, _ = time.Parse(time.RFC3339, t.LastPostedAt)
		}
		out = append(out, model.Topic{
			Site:        model.SiteNodeLoc,
			ExternalID:  fmt.Sprintf("%d", t.ID),
			CategoryID:  t.CategoryID,
			Title:       t.Title,
			URL:         fmt.Sprintf("%s/t/%s/%d", f.baseURL, t.Slug, t.ID),
			Excerpt:     t.Excerpt,
			PublishedAt: publishedAt,
			RawJSON:     "",
		})
	}
	return out, nil
}
