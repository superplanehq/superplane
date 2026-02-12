# Statuspage API - Quick Reference Guide

## Overview

The Statuspage API allows you to programmatically manage your status pages, incidents, components, subscribers, and metrics. This is the official Atlassian Statuspage API.

**Base URL:** `https://api.statuspage.io/v1/`  
**API Version:** 1.0.0  
**Protocol:** HTTPS (required)

## Authentication

All API requests require an API key passed via the `Authorization` header:

```
Authorization: OAuth YOUR_API_KEY
```

## Rate Limiting

- **Limit:** 1 request/second per API token (60-second rolling window)
- **Error Codes:** 420 or 429 indicate rate limit exceeded
- To increase limits, contact: https://support.atlassian.com/contact

## Content Types

The API accepts both JSON and form-urlencoded data. Ensure your `Content-Type` header matches:

**JSON (Recommended):**
```http
Content-Type: application/json
```

**Form Encoded:**
```http
Content-Type: application/x-www-form-urlencoded
```

---

# Core Resources

## 1. Pages

Your page profile drives basic settings including company name, notification preferences, and time zone.

### Get All Pages
```http
GET /pages
```

**Response:**
```json
[
  {
    "id": "kctbh9vrtdwd",
    "created_at": "2020-01-01T00:00:00.000Z",
    "updated_at": "2020-01-01T00:00:00.000Z",
    "name": "Example Status Page",
    "page_description": "Our status page",
    "headline": "Welcome",
    "branding": "basic",
    "subdomain": "example",
    "domain": "status.example.com",
    "url": "https://status.example.com",
    "support_url": "https://example.com/support",
    "hidden_from_search": false,
    "time_zone": "America/New_York"
  }
]
```

### Get Single Page
```http
GET /pages/{page_id}
```

### Update Page
```http
PATCH /pages/{page_id}
```

**Request Body:**
```json
{
  "page": {
    "name": "Updated Status Page",
    "domain": "status.example.com",
    "subdomain": "example",
    "url": "https://status.example.com"
  }
}
```

---

## 2. Incidents

Incidents represent critical events affecting your service. Three types:
- **Historical:** Past incidents submitted for record-keeping
- **Realtime:** Active incidents requiring immediate notification
- **Scheduled:** Planned maintenance windows

### Create an Incident
```http
POST /pages/{page_id}/incidents
```

**Request Body (Realtime Incident):**
```json
{
  "incident": {
    "name": "Database Connection Issues",
    "status": "investigating",
    "impact_override": "major",
    "body": "We are investigating reports of slow database queries.",
    "component_ids": ["8kbf7d35c070"],
    "components": {
      "8kbf7d35c070": "degraded_performance"
    }
  }
}
```

**Request Body (Scheduled Maintenance):**
```json
{
  "incident": {
    "name": "Database Upgrade",
    "status": "scheduled",
    "scheduled_for": "2026-02-15T02:00:00Z",
    "scheduled_until": "2026-02-15T04:00:00Z",
    "scheduled_remind_prior": true,
    "scheduled_auto_in_progress": true,
    "body": "We will be performing database maintenance.",
    "component_ids": ["8kbf7d35c070"],
    "components": {
      "8kbf7d35c070": "under_maintenance"
    }
  }
}
```

**Incident Status Values:**
- `investigating` - Issue identified, investigating cause
- `identified` - Issue identified, working on fix
- `monitoring` - Fix deployed, monitoring for stability
- `resolved` - Incident fully resolved
- `scheduled` - Maintenance is scheduled
- `in_progress` - Maintenance is in progress
- `completed` - Maintenance completed

**Impact Override Values:**
- `none` - No impact
- `minor` - Minor issues
- `major` - Major outage
- `critical` - Complete outage

### List Incidents
```http
GET /pages/{page_id}/incidents
GET /pages/{page_id}/incidents/unresolved
GET /pages/{page_id}/incidents/scheduled
GET /pages/{page_id}/incidents/active_maintenance
GET /pages/{page_id}/incidents/upcoming
```

**Query Parameters:**
- `q` - Text to search for in incident name
- `limit` - Number of results (default: 100)
- `page` - Page number for pagination

### Get Single Incident
```http
GET /pages/{page_id}/incidents/{incident_id}
```

### Update an Incident
```http
PATCH /pages/{page_id}/incidents/{incident_id}
```

**Request Body:**
```json
{
  "incident": {
    "status": "monitoring",
    "body": "We have deployed a fix and are monitoring the situation.",
    "component_ids": ["8kbf7d35c070"],
    "components": {
      "8kbf7d35c070": "operational"
    }
  }
}
```

### Delete an Incident
```http
DELETE /pages/{page_id}/incidents/{incident_id}
```

---

## 3. Incident Updates

Updates provide additional information about an ongoing incident.

### Create Incident Update
```http
POST /pages/{page_id}/incidents/{incident_id}/incident_updates
```

**Request Body:**
```json
{
  "incident_update": {
    "body": "The issue has been identified and a fix is being deployed.",
    "status": "identified"
  }
}
```

### Update Incident Update
```http
PATCH /pages/{page_id}/incidents/{incident_id}/incident_updates/{incident_update_id}
```

### List Incident Updates
```http
GET /pages/{page_id}/incidents/{incident_id}/incident_updates
```

---

## 4. Components

Components represent individual infrastructure pieces displayed on your status page.

### List Components
```http
GET /pages/{page_id}/components
```

**Response:**
```json
[
  {
    "id": "8kbf7d35c070",
    "page_id": "kctbh9vrtdwd",
    "group_id": null,
    "created_at": "2020-01-01T00:00:00.000Z",
    "updated_at": "2020-01-01T00:00:00.000Z",
    "group": false,
    "name": "API",
    "description": "Our REST API",
    "position": 1,
    "status": "operational",
    "showcase": true,
    "only_show_if_degraded": false
  }
]
```

### Update Component Status
```http
PATCH /pages/{page_id}/components/{component_id}
```

**Request Body:**
```json
{
  "component": {
    "status": "degraded_performance"
  }
}
```

**Component Status Values:**
- `operational` - Component is working normally
- `degraded_performance` - Component experiencing issues
- `partial_outage` - Component partially down
- `major_outage` - Component completely down
- `under_maintenance` - Component under maintenance

### Create Component
```http
POST /pages/{page_id}/components
```

**Request Body:**
```json
{
  "component": {
    "name": "Payment Gateway",
    "description": "Handles all payment processing",
    "status": "operational",
    "showcase": true
  }
}
```

---

## 5. Component Groups

Organize components into logical groups.

### Create Component Group
```http
POST /pages/{page_id}/component-groups
```

**Request Body:**
```json
{
  "component_group": {
    "name": "Backend Services",
    "description": "Core backend infrastructure",
    "components": ["8kbf7d35c070", "vtnh60py4yd7"]
  }
}
```

### List Component Groups
```http
GET /pages/{page_id}/component-groups
```

---

## 6. Subscribers

Subscribers receive notifications about incidents via email, SMS, Slack, Teams, or webhooks.

### Create Subscriber
```http
POST /pages/{page_id}/subscribers
```

**Email Subscriber:**
```json
{
  "subscriber": {
    "email": "user@example.com"
  }
}
```

**SMS Subscriber:**
```json
{
  "subscriber": {
    "phone_number": "+1234567890",
    "phone_country": "US"
  }
}
```

**Component Subscriber:**
```json
{
  "subscriber": {
    "email": "user@example.com",
    "component_ids": ["8kbf7d35c070"]
  }
}
```

### List Subscribers
```http
GET /pages/{page_id}/subscribers
```

**Query Parameters:**
- `type` - Filter by type: `email`, `sms`, `webhook`, `slack`, `integration`
- `state` - Filter by state: `active`, `quarantined`, `unconfirmed`
- `page` - Page number
- `limit` - Results per page (max: 100)

### Unsubscribe Subscriber
```http
DELETE /pages/{page_id}/subscribers/{subscriber_id}
```

### Bulk Operations
```http
POST /pages/{page_id}/subscribers/reactivate
POST /pages/{page_id}/subscribers/unsubscribe
POST /pages/{page_id}/subscribers/resend_confirmation
```

---

## 7. Metrics

Custom metrics display system performance data on your status page.

### Add Metric Data Point
```http
POST /pages/{page_id}/metrics/{metric_id}/data
```

**Request Body:**
```json
{
  "data": {
    "timestamp": 1676050800,
    "value": 142.5
  }
}
```

**Constraints:**
- Minimum: 1 data point every 5 minutes
- Data points cast to nearest 30s interval (max 10 per 5 min)
- Backfill supported up to 28 days

### Delete Metric Data
```http
DELETE /pages/{page_id}/metrics/{metric_id}/data/{metric_datum_id}
```

### List Metrics
```http
GET /pages/{page_id}/metrics
```

---

## 8. Templates

Templates provide pre-filled incident/maintenance content to save time.

### Create Template
```http
POST /pages/{page_id}/incident_templates
```

**Request Body:**
```json
{
  "incident_template": {
    "name": "Database Maintenance",
    "title": "Scheduled Database Maintenance",
    "body": "We will be performing routine database maintenance. Expect brief service interruptions.",
    "group_id": null,
    "should_tweet": false,
    "should_send_notifications": true
  }
}
```

### List Templates
```http
GET /pages/{page_id}/incident_templates
```

---

## 9. Postmortems

Detailed analysis published after incidents are resolved.

### Create Postmortem
```http
POST /pages/{page_id}/incidents/{incident_id}/postmortem
```

**Request Body:**
```json
{
  "postmortem": {
    "body": "## Root Cause\n\nThe incident was caused by...\n\n## Resolution\n\nWe resolved by...\n\n## Prevention\n\nTo prevent future occurrences..."
  }
}
```

### Publish Postmortem
```http
PUT /pages/{page_id}/incidents/{incident_id}/postmortem/publish
```

---

# Common Workflows

## Workflow 1: Create and Resolve an Incident

```python
# 1. Create incident
POST /pages/{page_id}/incidents
{
  "incident": {
    "name": "API Latency Issues",
    "status": "investigating",
    "impact_override": "major",
    "body": "We're investigating elevated API response times.",
    "component_ids": ["api_component_id"],
    "components": {
      "api_component_id": "degraded_performance"
    }
  }
}

# 2. Post update - issue identified
POST /pages/{page_id}/incidents/{incident_id}/incident_updates
{
  "incident_update": {
    "body": "Issue identified - database connection pool exhausted.",
    "status": "identified"
  }
}

# 3. Post update - fix deployed
POST /pages/{page_id}/incidents/{incident_id}/incident_updates
{
  "incident_update": {
    "body": "Fix deployed, monitoring for stability.",
    "status": "monitoring"
  }
}

# 4. Resolve incident
PATCH /pages/{page_id}/incidents/{incident_id}
{
  "incident": {
    "status": "resolved",
    "body": "All systems operational. Issue resolved.",
    "components": {
      "api_component_id": "operational"
    }
  }
}
```

## Workflow 2: Schedule Maintenance

```python
# 1. Create scheduled maintenance
POST /pages/{page_id}/incidents
{
  "incident": {
    "name": "Database Upgrade - Postgres 15",
    "status": "scheduled",
    "scheduled_for": "2026-02-20T03:00:00Z",
    "scheduled_until": "2026-02-20T05:00:00Z",
    "scheduled_remind_prior": true,
    "scheduled_auto_in_progress": true,
    "scheduled_auto_completed": true,
    "body": "We will upgrade our database to PostgreSQL 15.",
    "component_ids": ["database_component_id"],
    "components": {
      "database_component_id": "under_maintenance"
    }
  }
}

# Maintenance automatically transitions:
# - Reminder sent 60 min before scheduled_for
# - Status → "in_progress" at scheduled_for
# - Status → "completed" at scheduled_until
```

## Workflow 3: Bulk Component Status Update

```python
# Update multiple components during incident
PATCH /pages/{page_id}/incidents/{incident_id}
{
  "incident": {
    "status": "identified",
    "components": {
      "api_component_id": "major_outage",
      "database_component_id": "major_outage",
      "cache_component_id": "degraded_performance"
    }
  }
}
```

---

# Error Handling

## Error Response Format

```json
{
  "error": "Error message here",
  "status": 400
}
```

## Common HTTP Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request (validation error)
- `401` - Unauthorized (invalid API key)
- `404` - Not Found
- `420/429` - Rate Limit Exceeded
- `422` - Unprocessable Entity (validation failed)
- `500` - Internal Server Error

---

# Key Data Models

## Incident Object

```json
{
  "id": "string",
  "name": "string",
  "status": "investigating|identified|monitoring|resolved|scheduled|in_progress|completed",
  "impact": "none|minor|major|critical",
  "impact_override": "none|minor|major|critical",
  "created_at": "datetime",
  "updated_at": "datetime",
  "monitoring_at": "datetime",
  "resolved_at": "datetime",
  "shortlink": "string",
  "scheduled_for": "datetime",
  "scheduled_until": "datetime",
  "scheduled_remind_prior": "boolean",
  "scheduled_auto_in_progress": "boolean",
  "scheduled_auto_completed": "boolean",
  "metadata": {},
  "started_at": "datetime",
  "page_id": "string",
  "incident_updates": [],
  "components": [],
  "component_ids": []
}
```

## Component Object

```json
{
  "id": "string",
  "page_id": "string",
  "group_id": "string|null",
  "name": "string",
  "description": "string",
  "position": "integer",
  "status": "operational|degraded_performance|partial_outage|major_outage|under_maintenance",
  "showcase": "boolean",
  "only_show_if_degraded": "boolean",
  "automation_email": "string",
  "created_at": "datetime",
  "updated_at": "datetime",
  "start_date": "date"
}
```

## Subscriber Object

```json
{
  "id": "string",
  "skip_confirmation_notification": "boolean",
  "email": "string",
  "phone_number": "string",
  "phone_country": "string",
  "quarantined_at": "datetime|null",
  "created_at": "datetime",
  "mode": "email|sms|webhook|slack",
  "component_ids": [],
  "page_access_user": "string|null"
}
```

---

# Best Practices

1. **Always Set Impact Level**: Use `impact_override` to ensure correct severity is displayed
2. **Keep Users Updated**: Post incident updates every 15-30 minutes during active incidents
3. **Auto-resolve Components**: When resolving incidents, update component statuses back to `operational`
4. **Use Templates**: Create templates for common maintenance scenarios
5. **Monitor Rate Limits**: Implement exponential backoff for 429 errors
6. **Validate Before Posting**: Check component IDs exist before creating incidents
7. **Scheduled Maintenance**: Use `scheduled_remind_prior` for important maintenance
8. **Postmortems**: Add detailed postmortems for major incidents to build trust

---

# Additional Resources

- **API Documentation:** https://docs.statuspage.io/
- **Support:** https://support.atlassian.com/contact
- **Rate Limit Increases:** Contact support with your use case

---

**Last Updated:** February 2026  
**API Version:** 1.0.0
