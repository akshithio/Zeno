package hq

import (
	"log/slog"

	"github.com/internetarchive/Zeno/pkg/models"
	"github.com/internetarchive/gocrawlhq"
)

// SeencheckURLs sends a seencheck request to the crawl HQ for the given URLs.
func SeencheckURLs(URLsType models.URLType, URLs ...*models.URL) (seencheckedURLs []*models.URL, err error) {
	var discoveredURLs []gocrawlhq.URL

	for _, URL := range URLs {
		discoveredURLs = append(discoveredURLs, gocrawlhq.URL{
			Value: URL.String(),
			Type:  string(URLsType),
		})
	}

	outputURLs, err := globalHQ.client.Seencheck(discoveredURLs)
	if err != nil {
		slog.Error("error sending seencheck payload to crawl HQ", "err", err.Error())
		return URLs, err
	}

	if outputURLs != nil {
		for _, URL := range URLs {
			for _, outputURL := range outputURLs {
				if URL.String() == outputURL.Value {
					seencheckedURLs = append(seencheckedURLs, URL)
					break
				}
			}
		}
	}

	return seencheckedURLs, nil
}
