package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
)

// RecentTicketsCollector collects ticket metrics for the last 30 days
type RecentTicketsCollector struct {
	client *zendesk.Client
	status *prometheus.Desc
}

// NewRecentTicketsCollector creates a new RecentTicketsCollector
func NewRecentTicketsCollector(client *zendesk.Client) *RecentTicketsCollector {
	return &RecentTicketsCollector{
		client: client,
		status: prometheus.NewDesc(
			"zendesk_tickets_recent_status",
			"Number of tickets by status created in the last 30 days",
			[]string{"status"}, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *RecentTicketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.status
}

// Collect implements prometheus.Collector
func (c *RecentTicketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	timeRange := fmt.Sprintf("created>%s", thirtyDaysAgo.Format("2006-01-02"))

	statuses := []string{"new", "open", "pending", "solved", "closed"}
	for _, status := range statuses {
		searchQuery := fmt.Sprintf("%s status:%s type:ticket", timeRange, status)
		count, err := c.client.SearchCount(ctx, &zendesk.CountOptions{
			Query: searchQuery,
		})
		if err != nil {
			log.Printf("Error getting recent ticket count for status %s: %v", status, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			c.status,
			prometheus.GaugeValue,
			float64(count),
			status,
		)
	}
}
