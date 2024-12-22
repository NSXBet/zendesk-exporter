# Zendesk Exporter

Prometheus exporter for Zendesk metrics.

## Overview

This exporter exposes Zendesk metrics in Prometheus format, enabling monitoring of your Zendesk instance. It provides metrics about tickets, their statuses, tags, and custom fields.

## Collectors

Name | Description
---------|-------------
tickets | Detailed ticket metrics with status, priority, channel, type, tag, and custom field labels
recent_tickets | Simple ticket counts by status for the last 30 days
tags_tickets | Ticket counts by tags and status
custom_fields | Ticket counts by custom field values
all_time_tickets | Historical ticket metrics

## Prerequisites

- Zendesk Admin account or appropriate API access
- Zendesk API token

## Configuration

### Environment Variables

Name | Description
---------|-------------
ZENDESK_USERNAME | Username for Zendesk API
ZENDESK_API_TOKEN | API token for Zendesk API
ZENDESK_EMAIL | Email for Zendesk API

### Using Docker

```bash
docker compose up
```

## Exported Metrics

### Ticket Metrics

Name | Description | Labels
---------|-------------|--------
zendesk_tickets_count | Number of tickets created in the last 30 days | status, priority, channel, type, tag, custom_field
zendesk_tickets_total | Total number of tickets by status created in the last 30 days | status

### Recent Ticket Metrics

Name | Description | Labels
---------|-------------|--------
zendesk_tickets_recent_status_count | Number of tickets by status created in the last 30 days | status
zendesk_tickets_recent_status_total | Total number of tickets in the last 30 days | none

### Tag Metrics

Name | Description | Labels
---------|-------------|--------
zendesk_tickets_tags_count | Number of tickets by tag and status created in the last 30 days | tag, status
zendesk_tickets_tags_total | Total number of tickets with tags in the last 30 days | status

## License

Apache License 2.0