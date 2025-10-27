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
  const [isOpen, setIsOpen] = useState(true);
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

  const actions: Conversations.ContextAction[] = []; // Define context actions
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
