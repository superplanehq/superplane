import { BugIcon, CogIcon, RabbitIcon, ScanEye } from "lucide-react";
import { useState } from "react";
import { Conversations } from "./Conversations";
import { FloatingActionButton } from "./FloatingActionButton";

export function AiSidebar() {
  const state = useAiSidebarState();

  return (
    <>
      <FloatingActionButton onClick={state.openSidebar} />

      <Conversations
        isOpen={state.isOpen}
        onClose={state.closeSidebar}
        conversations={state.conversations}
        activeConversationId={state.activeConversationId}
        onSelectConversation={state.setActiveConversationId}
        onCreateConversation={state.createConvo}
        onSendMessage={state.sendMessage}
        contextActions={state.actions}
        contextAttachment={state.conversationContext!}
        maxWidth={1000}
      />
    </>
  );
}

function useAiSidebarState() {
  const [isOpen, setIsOpen] = useState(false);
  const [activeConversationId, setActiveConversationId] = useState<
    string | undefined
  >(undefined);
  const conversations: Conversations.Conversation[] = []; // Replace with actual conversations state

  function openSidebar() {
    setIsOpen(true);
  }

  function closeSidebar() {
    setIsOpen(false);
  }

  function createConvo() {
    // Implementation for creating a new conversation
  }

  function sendMessage(
    message: string,
    conversationId?: string | undefined
  ): Promise<void> {
    console.log("Sending message to conversation", conversationId, message);
    // Implementation for sending a message
    return Promise.resolve();
  }

  const actions: Conversations.ContextAction[] = [
    {
      id: "review-workflow",
      icon: <ScanEye size={24} className="text-sky-500" />,
      label: "Review this workflow",
      description: "Get AI to review your workflow for potential issues.",
    },
    {
      id: "optimize-steps",
      icon: <CogIcon size={24} className="text-green-500" />,
      label: "Optimize steps",
      description: "Suggest optimizations for the steps in this workflow.",
    },
    {
      id: "improve-performance",
      icon: <RabbitIcon size={24} className="text-purple-500" />,
      label: "Improve performance",
      description: "Get recommendations to enhance workflow performance.",
    },
    {
      id: "add-error-handling",
      icon: <BugIcon size={24} className="text-red-500" />,
      label: "Add error handling",
      description: "Suggest error handling mechanisms for this workflow.",
    },
  ];

  const conversationContext = null; // Define conversation context

  return {
    isOpen,
    openSidebar,
    closeSidebar,
    conversations,
    activeConversationId,
    setActiveConversationId,
    createConvo,
    sendMessage,
    actions,
    conversationContext,
  };
}
