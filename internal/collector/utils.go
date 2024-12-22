package collector

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/nukosuke/go-zendesk/zendesk"
)

// SearchResult holds the result of a status-based search
type SearchResult struct {
	Status string
	Items  []zendesk.Ticket
	Error  error
}

// StatusSearcher defines a function type that processes tickets for a specific status
type StatusSearcher func(status string, tickets []zendesk.Ticket) error

// SearchByStatus performs a parallel search across all statuses and processes results
func SearchByStatus(ctx context.Context, client *zendesk.Client, timeRange string, processor StatusSearcher) error {
	statuses := []string{"new", "open", "pending", "solved"} // omit closed status, too many tickets
	var wg sync.WaitGroup
	resultChan := make(chan SearchResult, len(statuses))

	// Process each status in parallel
	for _, status := range statuses {
		wg.Add(1)
		go func(status string) {
			defer wg.Done()

			tickets, err := searchTickets(ctx, client, status, timeRange)
			resultChan <- SearchResult{
				Status: status,
				Items:  tickets,
				Error:  err,
			}
		}(status)
	}

	// Close result channel after all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results as they come in
	for result := range resultChan {
		if result.Error != nil {
			log.Printf("Error searching tickets for status %s: %v", result.Status, result.Error)
			continue
		}
		// Always call processor with status, even if no tickets found
		if err := processor(result.Status, result.Items); err != nil {
			return fmt.Errorf("error processing tickets for status %s: %w", result.Status, err)
		}
	}

	return nil
}

// searchTickets performs the actual search for a specific status
func searchTickets(ctx context.Context, client *zendesk.Client, status, timeRange string) ([]zendesk.Ticket, error) {
	searchQuery := fmt.Sprintf("%s status:%s type:ticket", timeRange, status)
	opts := &zendesk.SearchOptions{
		PageOptions: zendesk.PageOptions{
			Page:    1,
			PerPage: 100, // Maximum allowed by Zendesk API
		},
		Query: searchQuery,
	}

	var tickets []zendesk.Ticket

	for {
		results, page, err := client.Search(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, item := range results.List() {
			if ticket, ok := item.(zendesk.Ticket); ok {
				tickets = append(tickets, ticket)
			}
		}

		if !page.HasNext() {
			break
		}
		opts.Page++
	}

	return tickets, nil
}

// isNumeric checks if a string represents a number
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
