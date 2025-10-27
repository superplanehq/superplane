import React from "react";
import { FloatingActionButton } from "../FloatingActionButton";
import { IconRobotFace } from "../icons";
import { genPerson } from "../utils/storybook/genPeople";
import { Conversations, useConversations } from "./index";

// Mock context data for a goal page
const mockGoalContext: Conversations.ContextAttachment = {
  id: "goal-q4-revenue",
  type: "goal",
  title: "Q4 Revenue Target: $500K",
  url: "/goals/q4-revenue",
};

// Context-aware actions for goal page
const mockContextActions: Conversations.ContextAction[] = [
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
  {
    id: "identify-blockers",
    label: "Identify blockers",
  },
];

const me = genPerson();

/**
 * Example component showing how to integrate the Conversations component
 * into a real application with AI integration and context-aware actions
 */
export function ConversationsExample() {
  const {
    conversations,
    activeConversationId,
    isOpen,
    isLoading,
    error,
    openConversations,
    closeConversations,
    createConversation,
    selectConversation,
    updateConversationTitle,
    sendMessage,
  } = useConversations({
    // Start with some example conversations for prototyping
    initialConversations: [
      {
        id: "conv-1",
        title: "Q4 Revenue Goal Analysis",
        messages: [
          {
            id: "msg-1",
            content: "Run action 'Evaluate goal definition'",
            timestamp: new Date(Date.now() - 300000),
            sender: "user",
          },
          {
            id: "msg-2",
            content: `**Goal Definition Analysis for "${mockGoalContext.title}"**\n\nStrengths:\n• Clear monetary target ($500K)\n• Specific timeframe (Q4)\n• Measurable outcome\n\nAreas for improvement:\n• Could benefit from more specific success metrics\n• Missing breakdown of how to achieve this target\n• No mention of responsible team members\n\nRecommendation: Consider adding 2-3 key milestones and assigning ownership to make this goal more actionable.`,
            timestamp: new Date(Date.now() - 280000),
            sender: "ai",
          },
        ],
        createdAt: new Date(Date.now() - 400000),
        updatedAt: new Date(Date.now() - 280000),
        context: mockGoalContext,
      },
      {
        id: "conv-2",
        title: "Weekly Status Check",
        messages: [
          {
            id: "msg-3",
            content: "Run action 'Summarize current status'",
            timestamp: new Date(Date.now() - 180000),
            sender: "user",
          },
          {
            id: "msg-4",
            content: `**Current Status Summary for "${mockGoalContext.title}"**\n\n📊 Progress: $187K achieved (37.4% of target)\n📅 Time remaining: 45 days in Q4\n⚠️  Risk level: Moderate`,
            timestamp: new Date(Date.now() - 160000),
            sender: "ai",
          },
        ],
        createdAt: new Date(Date.now() - 200000),
        updatedAt: new Date(Date.now() - 160000),
        context: mockGoalContext,
      },
      {
        id: "conv-3",
        title: "Blocker Review",
        messages: [
          {
            id: "msg-5",
            content: "Run action 'Identify blockers'",
            timestamp: new Date(Date.now() - 600000), // 10 minutes ago (earlier this week)
            sender: "user",
          },
        ],
        createdAt: new Date(Date.now() - 600000),
        updatedAt: new Date(Date.now() - 600000),
        context: mockGoalContext,
      },
    ],

    // Example AI integration with context-aware responses
    onSendToAI: async (message, _conversationHistory) => {
      // This would be replaced with actual AI service call
      await new Promise((resolve) => setTimeout(resolve, 1500)); // Simulate API delay

      let response = "";
      let actions: Conversations.MessageAction[] = [];

      // Context-aware responses based on action commands
      if (message.includes("Run action 'Evaluate goal definition'")) {
        response = `**Goal Definition Analysis for "${mockGoalContext.title}"**\n\nStrengths:\n• Clear monetary target ($500K)\n• Specific timeframe (Q4)\n• Measurable outcome\n\nAreas for improvement:\n• Could benefit from more specific success metrics\n• Missing breakdown of how to achieve this target\n• No mention of responsible team members\n\nRecommendation: Consider adding 2-3 key milestones and assigning ownership to make this goal more actionable.`;
        actions = [
          {
            id: "post-to-goal",
            label: "Post this analysis for discussion",
            variant: "primary",
            onClick: () => alert("This would post the analysis to the goal message board"),
          },
          {
            id: "schedule-review",
            label: "Do something else",
            variant: "secondary",
            onClick: () => alert("This demonstrates having more than one follow-up action"),
          },
        ];
      } else if (message.includes("Run action 'Summarize current status'")) {
        response = `**Current Status Summary for "${mockGoalContext.title}"**\n\n📊 Progress: $187K achieved (37.4% of target)\n📅 Time remaining: 45 days in Q4\n⚠️  Risk level: Moderate\n\nKey metrics:\n• Monthly run rate: $62K (need $104K avg for remaining months)\n• Pipeline: $245K in qualified opportunities\n• Team capacity: 85% utilized\n\nNext actions needed: Focus on converting pipeline opportunities and consider resource reallocation.`;
        actions = [
          {
            id: "update-goal",
            label: "Submit a goal check-in with this status",
            variant: "primary",
            onClick: () => alert("This would update the goal status"),
          },
        ];
      } else if (message.includes("Run action 'Are we on track?'")) {
        response = `**On-Track Analysis for "${mockGoalContext.title}"**\n\n🔴 **Currently behind target**\n\nCurrent trajectory: $420K projected (84% of goal)\nGap to close: $80K\n\nMain risks:\n• Q4 seasonality affecting close rates\n• Two key team members on vacation in December\n• Economic headwinds impacting deal sizes\n\nMitigation strategies:\n• Accelerate Q3 pipeline conversion\n• Consider temporary resource augmentation\n• Focus on higher-value opportunities`;
        actions = [
          {
            id: "create-action-plan",
            label: "Post this analysis for discussion",
            variant: "primary",
            onClick: () => alert("This would post the analysis to the goal message board"),
          },
          {
            id: "alert-stakeholders",
            label: "Alert champion",
            variant: "secondary",
            onClick: () => alert("This would send a DM to the champion"),
          },
        ];
      } else if (message.includes("Run action 'Identify blockers'")) {
        response = `**Blocker Analysis for "${mockGoalContext.title}"**\n\n🚫 **Top 3 Blockers:**\n\n1. **Lead qualification bottleneck**\n   - Marketing leads have 23% lower conversion rate\n   - Sales team spending too much time on unqualified prospects\n\n2. **Pricing pressure**\n   - Average deal size down 15% from Q3\n   - Competitors offering aggressive discounts\n\n3. **Resource constraints**\n   - Implementation team at capacity\n   - Causing longer sales cycles\n\nImmediate actions: Improve lead scoring, review pricing strategy, consider implementation partnerships.`;
        actions = [
          {
            id: "assign-blockers",
            label: "Post this analysis for discussion",
            variant: "primary",
            onClick: () => alert("This would post the analysis to the goal message board"),
          },
        ];
      } else {
        // Generic responses for other messages
        if (message.toLowerCase().includes("project structure")) {
          response =
            "Your project appears to be well-organized with a clear component structure. I can see you're using TurboUI with components like Modal, Avatar, Button, and many others. Would you like me to analyze any specific aspect of the architecture?";
        } else if (message.toLowerCase().includes("component")) {
          response =
            "I can help you with component development! Based on your TurboUI library, I notice you follow consistent patterns with TypeScript interfaces, proper prop definitions, and Tailwind CSS styling. What specific component would you like to work on?";
        } else {
          response = `I understand you're asking about: "${message}". As Alfred, your AI COO, I can help with goal analysis, project review, strategic decisions, and more. How can I assist you today?`;
        }
      }

      // Return response with actions if any
      return { content: response, actions };
    },

    // No persistence for prototypes - just use in-memory state
  });

  return (
    <div className="relative">
      {/* Main application content */}
      <div className="p-8">
        <h1 className="text-3xl font-bold mb-4">TurboUI Project</h1>
        <p className="text-content-dimmed mb-6">Your component library with AI assistance</p>

        <div className="space-y-4">
          <div className="bg-surface-highlight p-6 rounded-lg border border-surface-outline">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 bg-accent-base rounded-full flex items-center justify-center">
                <IconRobotFace size={20} className="text-white" />
              </div>
              <div className="flex-1">
                <h2 className="font-semibold mb-1">Alfred - Your AI COO</h2>
                <p className="text-sm text-content-dimmed mb-3">
                  I have access to your current goal "Q4 Revenue Target: $500K" and can help you with context-aware
                  analysis and recommendations.
                </p>
                <p className="text-xs text-content-dimmed">
                  💡 Use the "Ask Alfred" button in the bottom-right corner for AI assistance
                </p>
              </div>
            </div>
          </div>

          <div className="bg-surface-base border border-surface-outline p-4 rounded-lg">
            <h3 className="font-semibold mb-3">Current Context</h3>
            <div className="flex items-center gap-3 p-3 bg-surface-highlight rounded border border-surface-outline">
              <div className="w-2 h-2 bg-green-500 rounded-full" />
              <div>
                <div className="font-medium text-sm">{mockGoalContext.title}</div>
                <div className="text-xs text-content-dimmed">Goal • Active</div>
              </div>
            </div>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              <strong>Error:</strong> {error}
            </div>
          )}

          {isLoading && (
            <div className="bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded">AI is thinking...</div>
          )}
        </div>
      </div>

      {/* Floating Action Button for AI Assistant */}
      <FloatingActionButton
        icon={<IconRobotFace size={20} />}
        text="Ask Alfred"
        onClick={openConversations}
        label="Ask Alfred about this goal"
        variant="primary"
        position="bottom-right"
      />

      {/* Conversations overlay */}
      <Conversations
        me={me}
        isOpen={isOpen}
        onClose={closeConversations}
        conversations={conversations}
        activeConversationId={activeConversationId}
        onSelectConversation={selectConversation}
        onCreateConversation={createConversation}
        onSendMessage={(message, conversationId) => sendMessage(message, conversationId, mockGoalContext)}
        onUpdateConversationTitle={updateConversationTitle}
        contextActions={mockContextActions}
        contextAttachment={mockGoalContext}
      />
    </div>
  );
}

export default ConversationsExample;
