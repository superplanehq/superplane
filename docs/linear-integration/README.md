# Linear Integration for SuperPlane

This document outlines the proposed Linear integration architecture for SuperPlane, implementing issue #3041.

## Overview

The Linear integration enables SuperPlane workflows to interact with Linear's issue tracking and project management platform. It provides both webhook triggers for reactive workflows and actions for proactive issue management.

## Components

### Core Integration

- **Name**: `linear`
- **Authentication**: Personal API Token
- **Base URL**: `https://api.linear.app/graphql`

### Triggers

#### 1. On Issue Created (`linear.onIssueCreated`)

Triggers when a new issue is created in Linear workspace.

**Configuration Options:**
- `teamIds` (optional): Filter by specific teams
- `labelIds` (optional): Filter by specific labels

**Event Data:**
```json
{
  "id": "issue-abc123",
  "identifier": "ENG-42",
  "title": "Fix login bug",
  "description": "Users are unable to log in with OAuth",
  "url": "https://linear.app/company/issue/ENG-42",
  "priority": 2,
  "state": "Todo",
  "teamName": "Engineering",
  "teamKey": "ENG",
  "assigneeId": "user-123",
  "assigneeName": "John Doe",
  "createdAt": "2024-02-11T20:30:00Z"
}
```

### Actions

#### 1. Create Issue (`linear.createIssue`)

Creates a new issue in Linear with specified properties.

**Configuration Options:**
- `title` (required): Issue title
- `description` (optional): Issue description
- `teamId` (required): Target team ID
- `assigneeId` (optional): User to assign
- `labelIds` (optional): Labels to apply
- `priority` (optional): Priority level (0-4)

**Output:**
```json
{
  "success": true,
  "id": "issue-def456",
  "identifier": "ENG-43",
  "title": "New issue created via SuperPlane",
  "url": "https://linear.app/company/issue/ENG-43"
}
```

## Implementation Architecture

### File Structure
```
pkg/integrations/linear/
├── linear.go          # Main integration registration
├── client.go          # Linear GraphQL API client
├── common.go          # Shared data structures
├── on_issue_created.go # Issue creation trigger
├── create_issue.go    # Issue creation action
└── linear_test.go     # Integration tests
```

### Key Features

1. **GraphQL API Integration**: Uses Linear's official GraphQL API for all operations
2. **Webhook Security**: HMAC-SHA256 signature verification for webhook payloads
3. **Resource Management**: Automatic fetching of teams, labels, and users for UI dropdowns
4. **Error Handling**: Comprehensive error handling with meaningful messages
5. **Filtering**: Advanced filtering capabilities for triggers
6. **Metadata Caching**: Caches team and label metadata for efficient configuration

### Authentication

Linear integration uses Personal API tokens:

1. User navigates to Linear Settings → API → Personal API keys
2. Creates new API key with descriptive name
3. Enters token in SuperPlane configuration
4. Integration validates token and fetches workspace metadata

### Webhook Setup

For the `OnIssueCreated` trigger:

1. Integration automatically creates Linear webhook on setup
2. Webhook points to SuperPlane's webhook endpoint
3. Payload includes HMAC signature for security
4. Webhook is automatically cleaned up when trigger is removed

## Benefits

- **Seamless Integration**: Native Linear experience within SuperPlane
- **Real-time Triggers**: Instant workflow execution on Linear events
- **Rich Metadata**: Access to full issue context and workspace information
- **Security**: Token-based authentication with webhook signature verification
- **Scalability**: GraphQL API provides efficient data fetching

## Use Cases

### Automation
- Auto-assign issues based on labels or team
- Create GitHub issues for bugs marked as "external"
- Send Slack notifications for high-priority issues

### Synchronization
- Sync Linear issues to external project management tools
- Update external databases when issue status changes
- Mirror issue data to analytics platforms

### Notifications
- Send email alerts for critical bugs
- Post to Discord when new features are requested
- Update status pages when incidents are created

## Future Enhancements

1. **Additional Triggers**:
   - Issue Updated
   - Issue Completed
   - Issue Assigned
   - Comment Added

2. **Additional Actions**:
   - Update Issue
   - Add Comment
   - Assign User
   - Apply Labels

3. **Advanced Features**:
   - Bulk operations
   - Custom field support
   - Project milestone tracking
   - Cycle management

## Implementation Status

- [x] Architecture designed
- [x] API client structure defined  
- [x] Component interfaces documented
- [ ] Core integration implemented
- [ ] GraphQL client implemented
- [ ] Webhook handlers implemented
- [ ] Tests implemented
- [ ] Documentation completed

This integration follows SuperPlane's established patterns and provides a solid foundation for Linear workspace automation.