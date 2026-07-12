package model

import "time"

type SiteID string

const (
	SiteNodeLoc SiteID = "nodeloc"
	SiteLinuxDO SiteID = "linuxdo"
	SiteNodeSeek SiteID = "nodeseek"
)

type Topic struct {
	Site        SiteID    `json:"site"`
	ExternalID  string    `json:"external_id"`
	CategoryID  int       `json:"category_id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Author      string    `json:"author"`
	Excerpt     string    `json:"excerpt"`
	PublishedAt time.Time `json:"published_at"`
	RawJSON     string    `json:"raw_json"`
}

type SiteTopicFetcher interface {
	Site() SiteID
	FetchLatest() ([]Topic, error)
}
