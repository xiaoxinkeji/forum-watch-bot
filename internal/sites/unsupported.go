package sites

import (
	"fmt"
	"github.com/xiaoxinkeji/forum-watch-bot/internal/model"
)

type UnsupportedFetcher struct {
	site   model.SiteID
	reason string
}

func NewUnsupportedFetcher(site model.SiteID, reason string) *UnsupportedFetcher {
	return &UnsupportedFetcher{site: site, reason: reason}
}

func (f *UnsupportedFetcher) Site() model.SiteID { return f.site }

func (f *UnsupportedFetcher) FetchLatest() ([]model.Topic, error) {
	return nil, fmt.Errorf("site %s not available yet: %s", f.site, f.reason)
}
