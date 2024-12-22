package collector

import (
	"context"

	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
)

// TicketCollector collects Zendesk ticket metrics
type TicketCollector struct {
	client       *zendesk.Client
	ticketsTotal *prometheus.Desc
}

// NewTicketCollector creates a new TicketCollector
func NewTicketCollector(client *zendesk.Client) *TicketCollector {
	return &TicketCollector{
		client: client,
		ticketsTotal: prometheus.NewDesc(
			"zendesk_tickets_count",
			"Total number of tickets in Zendesk",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *TicketCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ticketsTotal
}

// Collect implements prometheus.Collector
func (c *TicketCollector) Collect(ch chan<- prometheus.Metric) {
	count, err := c.getTicketCount()
	if err != nil {
		// If there's an error, we'll report 0
		ch <- prometheus.MustNewConstMetric(
			c.ticketsTotal,
			prometheus.GaugeValue,
			0,
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.ticketsTotal,
		prometheus.GaugeValue,
		float64(count),
	)
}

func (c *TicketCollector) getTicketCount() (int64, error) {
	tickets, _, err := c.client.Tickets.List(context.Background(), &zendesk.ListTicketsOptions{})
	if err != nil {
		return 0, err
	}
	return int64(len(tickets)), nil
}
