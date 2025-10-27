import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";

import { genPerson } from "../utils/storybook/genPeople";
import { ConversationsExample } from "./ConversationsExample";
import { Conversations } from "./index";

const meta: Meta<typeof Conversations> = {
  title: "Components/Conversations",
  component: Conversations,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock context data
const mockGoalContext: Conversations.ContextAttachment = {
  id: "goal-q4-revenue",
  type: "goal",
  title: "Q4 Revenue Target: $500K",
  url: "/goals/q4-revenue",
};

// Context actions
const mockGoalActions: Conversations.ContextAction[] = [
  {
    id: "evaluate-definition",
    label: "Evaluate goal definition",
  },
  {
    id: "summarize-status",
    label: "Summarize current status",
  },
  {
    id: "on-track-analysis",
    label: "Are we on track?",
  },
];

const me = genPerson();

// Template component for stories
function ConversationsStory(args: any) {
  const [isOpen, setIsOpen] = useState(args.isOpen);
  const [conversations, setConversations] = useState<Conversations.Conversation[]>(args.conversations || []);
  const [activeConversationId, setActiveConversationId] = useState<string | undefined>(args.activeConversationId);

  const handleSendMessage = async (message: string, conversationId?: string) => {
    // Simulate AI response delay
    await new Promise((resolve) => setTimeout(resolve, 1500));

    const newMessage: Conversations.Message = {
      id: `msg-${Date.now()}`,
      content: message,
      timestamp: new Date(),
      sender: "user",
    };

    let aiResponseContent = `I understand you said: "${message}". This is a simulated AI response for the Storybook demo.`;
    let aiResponseActions: Conversations.MessageAction[] = [];

    // Add context-aware responses for demo
    if (message.includes("evaluate") || message.includes("definition")) {
      aiResponseContent = `**Analysis Complete**\n\nI've evaluated the request: "${message}"\n\nThis appears to be a context-aware action that would normally provide detailed analysis based on the attached goal or project data.`;
      aiResponseActions = [
        {
          id: "demo-action-1",
          label: "Take follow-up action",
          onClick: () => alert("This would perform a follow-up action"),
        },
        {
          id: "demo-action-2",
          label: "Share analysis",
          onClick: () => alert("This would share the analysis"),
        },
      ];
    }

    const aiResponse: Conversations.Message = {
      id: `msg-${Date.now() + 1}`,
      content: aiResponseContent,
      timestamp: new Date(Date.now() + 1000),
      sender: "ai",
      actions: aiResponseActions.length > 0 ? aiResponseActions : undefined,
    };

    setConversations((prev) => {
      if (conversationId) {
        // Add to existing conversation
        return prev.map((conv) =>
          conv.id === conversationId
            ? {
                ...conv,
                messages: [...conv.messages, newMessage, aiResponse],
                updatedAt: new Date(),
              }
            : conv,
        );
      } else {
        // Create new conversation
        const newConv: Conversations.Conversation = {
          id: `conv-${Date.now()}`,
          title: "New Chat",
          messages: [newMessage, aiResponse],
          createdAt: new Date(),
          updatedAt: new Date(),
          context: args.contextAttachment,
        };
        setActiveConversationId(newConv.id);
        return [newConv, ...prev];
      }
    });
  };

  const handleCreateConversation = () => {
    setActiveConversationId(undefined);
  };

  const handleUpdateConversationTitle = (id: string, newTitle: string) => {
    setConversations((prev) =>
      prev.map((conv) => (conv.id === id ? { ...conv, title: newTitle, updatedAt: new Date() } : conv)),
    );
  };

  return (
    <div className="h-screen bg-surface-base">
      <div className="p-8">
        <h1 className="text-2xl font-bold mb-4">Conversations Component Demo</h1>
        <p className="text-content-dimmed mb-4">This component provides an AI chat interface for project assistance.</p>
        <button
          onClick={() => setIsOpen(true)}
          className="px-4 py-2 bg-accent-base text-white rounded hover:bg-accent-hover transition-colors"
        >
          Open Conversations
        </button>
      </div>

      <Conversations
        me={me}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        conversations={conversations}
        activeConversationId={activeConversationId}
        onSelectConversation={setActiveConversationId}
        onCreateConversation={handleCreateConversation}
        onUpdateConversationTitle={handleUpdateConversationTitle}
        onSendMessage={handleSendMessage}
        contextActions={args.contextActions}
        contextAttachment={args.contextAttachment}
        initialWidth={args.initialWidth}
        minWidth={args.minWidth}
        maxWidth={args.maxWidth}
      />
    </div>
  );
}

// Cleaned up stories - keeping only the most useful ones

export const EmptyState: Story = {
  render: () => (
    <ConversationsStory
      isOpen={true}
      conversations={[]}
      contextActions={mockGoalActions}
      contextAttachment={mockGoalContext}
    />
  ),
};

// Removed ResizablePanel and GoalPageSimulation - Full Example covers everything

export const FullExample: Story = {
  render: () => <ConversationsExample />,
};
