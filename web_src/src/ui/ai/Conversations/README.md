# Conversations Component

A resizable AI chat interface component for project assistance and code review. The component appears as a sliding panel on the right side of the screen with support for multiple conversations, message history, and AI integration.

## Features

- **Resizable Panel**: Drag the left edge to resize the panel width
- **Multiple Conversations**: Support for managing multiple conversation threads
- **Message History**: Persistent conversation storage and retrieval
- **AI Integration**: Customizable AI service integration
- **Keyboard Shortcuts**: ESC to close, Enter to send messages
- **Responsive Design**: Adapts to different screen sizes
- **Accessibility**: Proper ARIA labels and keyboard navigation

## Basic Usage

```tsx
import { Conversations, useConversations } from "turboui";

function MyApp() {
  const {
    conversations,
    activeConversationId,
    isOpen,
    openConversations,
    closeConversations,
    createConversation,
    selectConversation,
    sendMessage,
  } = useConversations({
    onSendToAI: async (message, conversationHistory) => {
      // Your AI service integration
      const response = await myAIService.chat(message, conversationHistory);
      return response;
    },
  });

  return (
    <div>
      {/* Your app content */}
      <button onClick={openConversations}>Open AI Assistant</button>

      <Conversations
        isOpen={isOpen}
        onClose={closeConversations}
        conversations={conversations}
        activeConversationId={activeConversationId}
        onSelectConversation={selectConversation}
        onCreateConversation={createConversation}
        onSendMessage={sendMessage}
      />
    </div>
  );
}
```

## Advanced Usage with Custom Settings

```tsx
<Conversations
  isOpen={isOpen}
  onClose={closeConversations}
  conversations={conversations}
  activeConversationId={activeConversationId}
  onSelectConversation={selectConversation}
  onCreateConversation={createConversation}
  onSendMessage={sendMessage}
  initialWidth={450}
  minWidth={320}
  maxWidth={600}
/>
```

## Props

### Conversations Component

| Prop                   | Type                                                          | Default | Description                               |
| ---------------------- | ------------------------------------------------------------- | ------- | ----------------------------------------- |
| `isOpen`               | `boolean`                                                     | -       | Whether the conversations panel is open   |
| `onClose`              | `() => void`                                                  | -       | Called when the panel should close        |
| `onSendMessage`        | `(message: string, conversationId?: string) => Promise<void>` | -       | Called when a new message is sent         |
| `conversations`        | `Conversation[]`                                              | `[]`    | List of previous conversations            |
| `activeConversationId` | `string`                                                      | -       | Currently active conversation ID          |
| `onSelectConversation` | `(conversationId: string) => void`                            | -       | Called when a conversation is selected    |
| `onCreateConversation` | `() => void`                                                  | -       | Called when a new conversation is created |
| `initialWidth`         | `number`                                                      | `384`   | Initial width of the panel in pixels      |
| `minWidth`             | `number`                                                      | `320`   | Minimum width of the panel in pixels      |
| `maxWidth`             | `number`                                                      | `600`   | Maximum width of the panel in pixels      |

### useConversations Hook

| Option                 | Type                                                                    | Description                             |
| ---------------------- | ----------------------------------------------------------------------- | --------------------------------------- |
| `onSendToAI`           | `(message: string, conversationHistory?: Message[]) => Promise<string>` | Function to send messages to AI service |
| `onSaveConversation`   | `(conversation: Conversation) => Promise<void>`                         | Function to save conversations          |
| `onLoadConversations`  | `() => Promise<Conversation[]>`                                         | Function to load conversations          |
| `initialConversations` | `Conversation[]`                                                        | Initial conversations to load           |

## Types

```tsx
interface Message {
  id: string;
  content: string;
  timestamp: Date;
  sender: "user" | "ai";
}

interface Conversation {
  id: string;
  title: string;
  messages: Message[];
  createdAt: Date;
  updatedAt: Date;
}
```

## AI Integration Example

```tsx
const { ... } = useConversations({
  onSendToAI: async (message, conversationHistory) => {
    // Example with OpenAI
    const response = await openai.chat.completions.create({
      model: "gpt-4",
      messages: [
        { role: "system", content: "You are a helpful coding assistant." },
        ...conversationHistory?.map(msg => ({
          role: msg.sender === "user" ? "user" : "assistant",
          content: msg.content
        })) || [],
        { role: "user", content: message }
      ]
    });

    return response.choices[0].message.content;
  },

  // Persist conversations to localStorage
  onSaveConversation: async (conversation) => {
    const saved = JSON.parse(localStorage.getItem("conversations") || "[]");
    const updated = saved.filter(c => c.id !== conversation.id);
    updated.unshift(conversation);
    localStorage.setItem("conversations", JSON.stringify(updated.slice(0, 10)));
  },

  onLoadConversations: async () => {
    const saved = localStorage.getItem("conversations");
    return saved ? JSON.parse(saved) : [];
  },
});
```

## Styling

The component uses Tailwind CSS classes and follows the TurboUI design system:

- `surface-base` - Background color
- `surface-outline` - Border color
- `surface-highlight` - Hover states
- `content-accent` - Primary text
- `content-dimmed` - Secondary text
- `accent-base` - Action buttons

## Keyboard Shortcuts

- **ESC** - Close the conversations panel
- **Enter** - Send message (in input field)
- **Shift+Enter** - New line (in input field)

## Resize Functionality

The panel can be resized by dragging the left edge:

- Visual resize handle with dots indicator
- Smooth resizing with constraints
- Prevents text selection during resize
- Cursor changes to indicate resize mode

## Accessibility

- Proper ARIA labels and roles
- Keyboard navigation support
- Focus management
- Screen reader friendly message structure
