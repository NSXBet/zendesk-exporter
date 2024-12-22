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

// CustomFieldsCollector collects ticket custom field metrics for the last 30 days
type CustomFieldsCollector struct {
	client *zendesk.Client
	fields *prometheus.Desc
	total  *prometheus.Desc
}

// NewCustomFieldsCollector creates a new CustomFieldsCollector
func NewCustomFieldsCollector(client *zendesk.Client) *CustomFieldsCollector {
	return &CustomFieldsCollector{
		client: client,
		fields: prometheus.NewDesc(
			"zendesk_tickets_custom_fields_count",
			"Number of tickets by custom field value (excluding numeric values) and status created in the last 30 days",
			[]string{"field_value", "status"}, nil,
		),
		total: prometheus.NewDesc(
			"zendesk_tickets_custom_fields_total",
			"Total number of tickets with non-numeric custom fields in the last 30 days",
			[]string{"status"}, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *CustomFieldsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fields
	ch <- c.total
}

// Collect implements prometheus.Collector
func (c *CustomFieldsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	timeRange := fmt.Sprintf("created>%s", thirtyDaysAgo.Format("2006-01-02"))

	type statusMetrics struct {
		fieldValues map[string]float64 // field_value -> count
		total       int64
	}
	metrics := make(map[string]*statusMetrics)
	var mu sync.Mutex

	// Process tickets for each status
	err := SearchByStatus(ctx, c.client, timeRange, func(status string, tickets []zendesk.Ticket) error {
		fieldValues := make(map[string]float64)
		var statusTotal int64

		for _, ticket := range tickets {
			if len(ticket.CustomFields) > 0 {
				hasCustomField := false
				for _, field := range ticket.CustomFields {
					if field.Value != nil && field.Value != "" {
						fieldValue := fmt.Sprintf("%v", field.Value)
						// Skip if the value is numeric
						if !isNumeric(fieldValue) {
							fieldValues[fieldValue]++
							hasCustomField = true
						}
					}
				}
				if hasCustomField {
					statusTotal++
				}
			}
		}

		mu.Lock()
		metrics[status] = &statusMetrics{
			fieldValues: fieldValues,
			total:       statusTotal,
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
		// Send total tickets with custom fields for this status
		ch <- prometheus.MustNewConstMetric(
			c.total,
			prometheus.GaugeValue,
			float64(statusMetric.total),
			status,
		)

		// Send metrics for each field value in this status
		for fieldValue, count := range statusMetric.fieldValues {
			ch <- prometheus.MustNewConstMetric(
				c.fields,
				prometheus.GaugeValue,
				count,
				fieldValue,
				status,
			)
		}
	}

	// Log summary
	var totalWithFields int64
	uniqueValues := make(map[string]bool)
	for _, sm := range metrics {
		totalWithFields += sm.total
		for value := range sm.fieldValues {
			uniqueValues[value] = true
		}
	}
	log.Printf("Collected tickets with non-numeric custom fields: %d, unique field values: %d", totalWithFields, len(uniqueValues))
}
