import { BugIcon, CogIcon, RabbitIcon, ScanEye } from "lucide-react";
import { useState } from "react";
import { sleep } from "../CanvasPage/storybooks/useSimulation";
import { Conversations } from "./Conversations";
import { FloatingActionButton } from "./FloatingActionButton";

interface AiSidebarState {
  showNotifications: boolean;
  notificationMessage?: string;
}

export function AiSidebar(props: AiSidebarState) {
  const state = useAiSidebarState();

  return (
    <>
      <FloatingActionButton
        onClick={state.openSidebar}
        showNotification={props.showNotifications}
        notificationMessage={props.notificationMessage}
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
  const [conversations, setConversations] = useState<
    Conversations.Conversation[]
  >([]);

  function openSidebar() {
    setIsOpen(true);
  }

  function closeSidebar() {
    setIsOpen(false);
  }

  async function createConvo(action: Conversations.ContextAction | null) {
    if (!action) return;

    const id = `convo-${conversations.length + 1}`;

    setConversations((prev) => {
      const newConvo: Conversations.Conversation = {
        id: id,
        title: action.label,
        messages: [
          {
            id: `msg-1`,
            content: `Reviewing your workflow and preparing suggestions...`,
            timestamp: new Date(),
            sender: "ai",
            status: "done",
            actions: [],
          },
        ],
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      return [...prev, newConvo];
    });

    setActiveConversationId(id);

    await sleep(3000);

    setConversations((prev) =>
      prev.map((convo) => {
        if (convo.id === id) {
          return {
            ...convo,
            messages: [
              ...convo.messages,
              {
                id: `msg-2`,
                timestamp: new Date(),
                content: "",
                sender: "ai",
                status: "done",
                actions: [],
              },
            ],
            updatedAt: new Date(),
          };
        }
        return convo;
      })
    );

    const msg = [
      "Here are some suggestions to improve your workflow:",
      "",
      "1. Set a filter on the GitHub trigger to select only pushes from the main branch. This will reduce unnecessary workflow runs.",
      "2. Add a caching step for dependencies in the CI job to speed up build times.",
      "3. Include error handling in the deployment step to manage potential failures gracefully.",
      "4. Optimize the order of steps to run tests before building the application, saving resources if tests fail.",
      "",
      "Let me know if you'd like help implementing any of these!",
    ].join("\n");

    // Simulate typing the message word by word
    const words = msg.split(" ");
    for (let i = 0; i < words.length; i++) {
      await sleep(50);
      const partialMessage = words.slice(0, i + 1).join(" ");

      setConversations((prev) =>
        prev.map((convo) => {
          if (convo.id === id) {
            const updatedMessages = [...convo.messages];
            updatedMessages[updatedMessages.length - 1] = {
              ...updatedMessages[updatedMessages.length - 1],
              content: partialMessage,
            };
            return {
              ...convo,
              messages: updatedMessages,
              updatedAt: new Date(),
            };
          }
          return convo;
        })
      );
    }
  }

  function sendMessage(
    message: string,
    conversationId?: string | undefined
  ): Promise<void> {
    console.log("Sending message to conversation", conversationId, message);
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
