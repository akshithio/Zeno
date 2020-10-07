package crawl

import (
	"strings"

	"github.com/CorentinB/Zeno/internal/pkg/frontier"

	log "github.com/sirupsen/logrus"
)

// Worker archive the items!
func (c *Crawl) Worker(item *frontier.Item) {
	// Capture the page
	outlinks, err := c.capture(item)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Debug(item.URL.String())
		return
	}

	// Send the outlinks to the pool of workers
	if item.Hop < c.MaxHops {
		for _, outlink := range outlinks {
			outlink := outlink
			if c.DomainsCrawl && strings.Contains(item.Host, outlink.Host) && item.Hop == 0 {
				newItem := frontier.NewItem(&outlink, item, 0)
				if c.UseKafka && len(c.KafkaOutlinksTopic) > 0 {
					c.KafkaProducerChannel <- newItem
				} else {
					c.Frontier.PushChan <- newItem
				}
			} else {
				newItem := frontier.NewItem(&outlink, item, item.Hop+1)
				if c.UseKafka && len(c.KafkaOutlinksTopic) > 0 {
					c.KafkaProducerChannel <- newItem
				} else {
					c.Frontier.PushChan <- newItem
				}
			}
		}
	}
}
