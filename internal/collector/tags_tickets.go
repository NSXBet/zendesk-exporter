package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
)

// TagsTicketsCollector collects ticket tag metrics for the last 30 days
type TagsTicketsCollector struct {
	client *zendesk.Client
	tags   *prometheus.Desc
	total  *prometheus.Desc
}

// NewTagsTicketsCollector creates a new TagsTicketsCollector
func NewTagsTicketsCollector(client *zendesk.Client) *TagsTicketsCollector {
	return &TagsTicketsCollector{
		client: client,
		tags: prometheus.NewDesc(
			"zendesk_tickets_tags_count",
			"Number of tickets by tag and status created in the last 30 days",
			[]string{"tag", "status"}, nil,
		),
		total: prometheus.NewDesc(
			"zendesk_tickets_tags_total",
			"Total number of tickets with tags in the last 30 days",
			[]string{"status"}, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *TagsTicketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tags
	ch <- c.total
}

// Collect implements prometheus.Collector
func (c *TagsTicketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	timeRange := fmt.Sprintf("created>%s", thirtyDaysAgo.Format("2006-01-02"))

	type statusMetrics struct {
		tags  map[string]float64
		total int64
	}
	metrics := make(map[string]*statusMetrics)
	var mu sync.Mutex

	// Process tickets for each status
	err := SearchByStatus(ctx, c.client, timeRange, func(status string, tickets []zendesk.Ticket) error {
		statusTags := make(map[string]float64)
		var statusTotal int64

		for _, ticket := range tickets {
			if len(ticket.Tags) > 0 {
				for _, tag := range ticket.Tags {
					statusTags[tag]++
				}
				statusTotal++
			}
		}

		mu.Lock()
		metrics[status] = &statusMetrics{
			tags:  statusTags,
			total: statusTotal,
		}
		mu.Unlock()

		return nil
	})

	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}

	// Send all metrics at once
	for status, statusMetric := range metrics {
		// Send total tickets with tags for this status
		ch <- prometheus.MustNewConstMetric(
			c.total,
			prometheus.GaugeValue,
			float64(statusMetric.total),
			status,
		)

		// Send metrics for each tag in this status
		for tag, count := range statusMetric.tags {
			ch <- prometheus.MustNewConstMetric(
				c.tags,
				prometheus.GaugeValue,
				count,
				tag,
				status,
			)
		}
	}

	// Log summary
	var totalTagged int64
	uniqueTags := make(map[string]bool)
	for _, sm := range metrics {
		totalTagged += sm.total
		for tag := range sm.tags {
			uniqueTags[tag] = true
		}
	}
	log.Printf("Collected tickets with tags: %d, unique tags: %d", totalTagged, len(uniqueTags))
}
