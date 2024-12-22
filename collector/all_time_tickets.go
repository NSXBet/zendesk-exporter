package collector

import (
	"context"
	"log"

	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
)

// AllTimeTicketsCollector collects total number of tickets across all time
type AllTimeTicketsCollector struct {
	client *zendesk.Client
	total  *prometheus.Desc
}

// NewAllTimeTicketsCollector creates a new AllTimeTicketsCollector
func NewAllTimeTicketsCollector(client *zendesk.Client) *AllTimeTicketsCollector {
	return &AllTimeTicketsCollector{
		client: client,
		total: prometheus.NewDesc(
			"zendesk_tickets_all_time_total",
			"Total number of tickets in Zendesk across all time",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *AllTimeTicketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.total
}

// Collect implements prometheus.Collector
func (c *AllTimeTicketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	count, err := c.client.SearchCount(ctx, &zendesk.CountOptions{
		Query: "type:ticket",
	})
	if err != nil {
		log.Printf("Error getting all-time ticket count: %v", err)
		ch <- prometheus.MustNewConstMetric(
			c.total,
			prometheus.GaugeValue,
			0,
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.total,
		prometheus.GaugeValue,
		float64(count),
	)
}
