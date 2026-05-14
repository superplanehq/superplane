# Agent Rich UI Widgets

This document describes the rich UI widgets implemented for the Agent sidebar chat.

## Overview

The Agent sidebar now supports rich interactive widgets that render inside chat messages using a custom `:::type ... :::` block syntax. The parser splits messages into segments, and each segment is rendered with the appropriate component.

## Implementation

### Parser (`widgets/parser.ts`)

Parses agent markdown content and splits it into segments:
- Regular markdown text (rendered by ReactMarkdown)
- Custom blocks using `:::type ... :::` syntax

Supported block types:
- `buttons` - horizontal button group
- `confirm` - confirmation dialog with yes/no buttons
- `chart` - data visualization (line, bar, area, pie)
- `collapse` - collapsible section
- `steps` - vertical checklist with checkmarks/spinner
- `success` - success banner (green)
- `error` - error banner (red)

### Widget Components (`widgets/`)

Each widget has its own file:

- **ButtonsWidget.tsx** - Horizontal button group with violet outline style
- **ConfirmWidget.tsx** - Warning card with message and yes/no buttons
- **ChartWidget.tsx** - Data visualization using Recharts (line, bar, area, pie)
- **CollapseWidget.tsx** - Disclosure widget, collapsed by default
- **StepsWidget.tsx** - Vertical checklist with checkmarks and spinners
- **BannerWidget.tsx** - Success (green) or error (red) banner with icon

### RichMessage Component (`widgets/RichMessage.tsx`)

Main component that:
1. Parses the message content using `parseAgentMessage()`
2. Renders each segment with the appropriate widget or ReactMarkdown
3. Accepts an `onAction` callback for button/confirm clicks

### Integration

Updated `AgentSidebar/index.tsx`:
- Imported `RichMessage` component
- Replaced `AgentMarkdown` with `RichMessage` in `MessageRow`
- Added `handleAction` callback that sends user actions as new messages
- Removed unused `AgentMarkdown` component and related imports

### Storybook Stories (`widgets/RichMessage.stories.tsx`)

Comprehensive story file showcasing all widgets:
- Buttons example
- Chart examples (line, bar, area, pie)
- Mixed content (markdown + widgets)
- Confirmation dialog
- Steps in progress
- Success and error banners
- Collapsible sections
- Complex multi-widget message
- Pure markdown (no widgets)

## Usage Examples

### Buttons

```markdown
What would you like to do?

:::buttons
- Create new file
- Update existing file
- Delete file
:::
```

### Chart

```markdown
Build performance:

:::chart
{
  "type": "line",
  "data": [
    { "day": "Mon", "buildTime": 45 },
    { "day": "Tue", "buildTime": 52 }
  ],
  "xKey": "day",
  "yKeys": ["buildTime"]
}
:::
```

### Confirm

```markdown
:::confirm
message: This will remove all unused dependencies. Continue?
yes: Remove dependencies
no: Keep them
:::
```

### Steps

```markdown
:::steps
- [x] Building application
- [x] Running tests
- [ ] Deploying
:::
```

### Success Banner

```markdown
:::success
Deployment completed successfully!
:::
```

### Collapse

```markdown
:::collapse
title: View configuration
content: |
  apiVersion: v1
  kind: Service
:::
```

## Dependencies

- **recharts** (^2.15.0) - Added to package.json for chart rendering

## Running Storybook

```bash
cd web_src
npm run storybook
```

Navigate to http://localhost:6006 and view the "AgentSidebar/RichMessage" stories.

## Notes

- TypeScript compilation passes with no errors
- The implementation uses existing Tailwind classes and the Button component
- Charts have a fixed height of 200px with responsive width
- All widgets are compact and designed to fit inside chat bubbles
- The parser handles edge cases: no custom blocks, multiple blocks, blocks at any position
- Actions from buttons/confirms are sent as new chat messages via the existing mutation

## Future Enhancements

- Add syntax highlighting for code blocks (shiki/highlight.js)
- Add more chart types (scatter, radar)
- Add inline actions (e.g., copy code, download)
- Add progress bars and sliders
- Add file upload widgets
