# Approvals

Superplane supports flexible approval workflows for stage events. You can configure approval requirements that specify exactly who needs to approve before a stage can execute.

## Approval Requirements

For more sophisticated approval workflows, you can specify detailed requirements using the `from` field:

```yaml
name: production-deployment
conditions:
  - type: approval
    approval:
      from:
        # Require approval from specific user, no count is required
        - type: user
          id: "12345678-1234-1234-1234-123456789abc"
        
        # Require approval from user by username, no count is required
        - type: user
          name: "john.doe"
          
        # Require approvals from users with specific role
        - type: role
          name: "canvas_admin"
          count: 2
          
        # Require approval from users in specific group
        - type: group
          name: "security-team"
          count: 1
```

## Requirement Types

### User Requirements

You can require approvals from specific users in two ways:

**By User ID:**
```yaml
- type: user
  id: "12345678-1234-1234-1234-123456789abc"
```

**By Username:**
```yaml
- type: user
  name: "john.doe"
```

### Role Requirements

Require approvals from users with specific roles:

```yaml
- type: role
  name: "canvas_admin"  # Or canvas_owner, canvas_viewer, etc.
  count: 2
```

Default available roles:
- `canvas_owner` - Full control over the canvas
- `canvas_admin` - Can manage canvas resources and approve events
- `canvas_viewer` - Read-only access to canvas

### Group Requirements

Require approvals from users in specific groups:

```yaml
- type: group
  name: "security-team"
  count: 1
```

Groups are organization-level entities that contain multiple users.

## How Approval Validation Works

When a stage event is created and has approval conditions:

1. The event enters the "waiting" state with reason "approval"
2. Users with appropriate permissions can approve the event
3. The system checks all requirements defined in `from`
4. ALL requirements must be satisfied for the event to proceed
5. Once satisfied, the event moves to the next condition or begins execution

## Permission Requirements

To approve stage events, users need the `stageevent:approve` permission on the canvas. By default, this is granted to:
- Canvas owners
- Canvas admins