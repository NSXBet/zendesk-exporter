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

// RecentTicketsCollector collects ticket metrics for the last 30 days
type RecentTicketsCollector struct {
	client *zendesk.Client
	status *prometheus.Desc
	total  *prometheus.Desc
}

// NewRecentTicketsCollector creates a new RecentTicketsCollector
func NewRecentTicketsCollector(client *zendesk.Client) *RecentTicketsCollector {
	return &RecentTicketsCollector{
		client: client,
		status: prometheus.NewDesc(
			"zendesk_tickets_recent_status_count",
			"Number of tickets by status created in the last 30 days",
			[]string{"status"}, nil,
		),
		total: prometheus.NewDesc(
			"zendesk_tickets_recent_status_total",
			"Total number of tickets in the last 30 days",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *RecentTicketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.status
	ch <- c.total
}

// Collect implements prometheus.Collector
func (c *RecentTicketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	timeRange := fmt.Sprintf("created>%s", thirtyDaysAgo.Format("2006-01-02"))

	metrics := make(map[string]float64)
	var totalTickets int64
	var mu sync.Mutex

	// Process tickets for each status
	err := SearchByStatus(ctx, c.client, timeRange, func(status string, tickets []zendesk.Ticket) error {
		count := float64(len(tickets))

		mu.Lock()
		metrics[status] = count
		totalTickets += int64(count)
		mu.Unlock()

		return nil
	})

	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}

	// Send all metrics at once
	for status, count := range metrics {
		ch <- prometheus.MustNewConstMetric(
			c.status,
			prometheus.GaugeValue,
			count,
			status,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.total,
		prometheus.GaugeValue,
		float64(totalTickets),
	)

	log.Printf("Collected total tickets: %d, counts by status: %v", totalTickets, metrics)
}
