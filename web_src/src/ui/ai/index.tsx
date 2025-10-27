import { FloatingActionButton } from "./FloatingActionButton";

export function AiSidebar() {
  <>
    <FloatingActionButton
      icon={null}
      text="AI Assist"
      onClick={() => {}}
      label={""}
      variant="primary"
      position="bottom-right"
    />

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
      me={me!}
      maxWidth={1000}
    />
  </>;
}
