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

// TicketsCollector collects detailed ticket metrics for the last 30 days
type TicketsCollector struct {
	client  *zendesk.Client
	tickets *prometheus.Desc
	total   *prometheus.Desc
}

// NewTicketsCollector creates a new TicketsCollector
func NewTicketsCollector(client *zendesk.Client) *TicketsCollector {
	return &TicketsCollector{
		client: client,
		tickets: prometheus.NewDesc(
			"zendesk_tickets_count",
			"Number of tickets by status, priority, channel, type, tag, and custom field created in the last 30 days",
			[]string{"status", "priority", "channel", "type", "tag", "custom_field"}, nil,
		),
		total: prometheus.NewDesc(
			"zendesk_tickets_total",
			"Total number of tickets by status created in the last 30 days",
			[]string{"status"}, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *TicketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tickets
	ch <- c.total
}

// Collect implements prometheus.Collector
func (c *TicketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	timeRange := fmt.Sprintf("created>%s", thirtyDaysAgo.Format("2006-01-02"))

	// Initialize counts map
	counts := make(map[string]map[string]map[string]map[string]map[string]map[string]int) // status->priority->channel->type->tag->customfield->count
	statusTotals := make(map[string]int)                                                  // status->total
	var mu sync.Mutex

	// Process tickets for each status
	err := SearchByStatus(ctx, c.client, timeRange, func(status string, tickets []zendesk.Ticket) error {
		localCounts := make(map[string]map[string]map[string]map[string]map[string]map[string]int)
		localTotal := 0

		for _, ticket := range tickets {
			localTotal++

			channel := "unknown"
			if ticket.Via != nil {
				channel = ticket.Via.Channel
			}

			priority := ticket.Priority
			if priority == "" {
				priority = "none"
			}

			ticketType := ticket.Type
			if ticketType == "" {
				ticketType = "none"
			}

			// Process tags
			tags := map[string]bool{"none": true}
			for _, tag := range ticket.Tags {
				if tag != "" {
					tags[tag] = true
					delete(tags, "none")
				}
			}

			// Process custom fields
			customFields := map[string]bool{"none": true}
			for _, field := range ticket.CustomFields {
				if field.Value != nil {
					value := fmt.Sprintf("%v", field.Value)
					if value != "" && !isNumeric(value) {
						customFields[value] = true
						delete(customFields, "none")
					}
				}
			}

			// Initialize and increment counters
			for tag := range tags {
				for customField := range customFields {
					if localCounts[status] == nil {
						localCounts[status] = make(map[string]map[string]map[string]map[string]map[string]int)
					}
					if localCounts[status][priority] == nil {
						localCounts[status][priority] = make(map[string]map[string]map[string]map[string]int)
					}
					if localCounts[status][priority][channel] == nil {
						localCounts[status][priority][channel] = make(map[string]map[string]map[string]int)
					}
					if localCounts[status][priority][channel][ticketType] == nil {
						localCounts[status][priority][channel][ticketType] = make(map[string]map[string]int)
					}
					if localCounts[status][priority][channel][ticketType][tag] == nil {
						localCounts[status][priority][channel][ticketType][tag] = make(map[string]int)
					}
					localCounts[status][priority][channel][ticketType][tag][customField]++
				}
			}
		}

		// Merge local counts into global counts
		mu.Lock()
		statusTotals[status] += localTotal
		for status, priorityMap := range localCounts {
			if counts[status] == nil {
				counts[status] = make(map[string]map[string]map[string]map[string]map[string]int)
			}
			for priority, channelMap := range priorityMap {
				if counts[status][priority] == nil {
					counts[status][priority] = make(map[string]map[string]map[string]map[string]int)
				}
				for channel, typeMap := range channelMap {
					if counts[status][priority][channel] == nil {
						counts[status][priority][channel] = make(map[string]map[string]map[string]int)
					}
					for ticketType, tagMap := range typeMap {
						if counts[status][priority][channel][ticketType] == nil {
							counts[status][priority][channel][ticketType] = make(map[string]map[string]int)
						}
						for tag, customFieldMap := range tagMap {
							if counts[status][priority][channel][ticketType][tag] == nil {
								counts[status][priority][channel][ticketType][tag] = make(map[string]int)
							}
							for customField, count := range customFieldMap {
								counts[status][priority][channel][ticketType][tag][customField] += count
							}
						}
					}
				}
			}
		}
		mu.Unlock()

		return nil
	})

	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}

	// Send total metrics first
	var totalTickets int
	for status, total := range statusTotals {
		ch <- prometheus.MustNewConstMetric(
			c.total,
			prometheus.GaugeValue,
			float64(total),
			status,
		)
		totalTickets += total
	}

	// Send detailed metrics
	var totalDetailedMetrics int
	for status, priorityMap := range counts {
		for priority, channelMap := range priorityMap {
			for channel, typeMap := range channelMap {
				for ticketType, tagMap := range typeMap {
					for tag, customFieldMap := range tagMap {
						for customField, count := range customFieldMap {
							ch <- prometheus.MustNewConstMetric(
								c.tickets,
								prometheus.GaugeValue,
								float64(count),
								status,
								priority,
								channel,
								ticketType,
								tag,
								customField,
							)
							totalDetailedMetrics++
						}
					}
				}
			}
		}
	}

	log.Printf("Collected %d total tickets across %d detailed metrics", totalTickets, totalDetailedMetrics)
}
