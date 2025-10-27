import {
  BotIcon,
  CheckIcon,
  CopyIcon,
  FileIcon,
  ListIcon,
  PaperclipIcon,
  PlusIcon,
  XIcon,
} from "lucide-react";

import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Button } from "../../button";

export namespace Conversations {
  export interface Message {
    id: string;
    content: string;
    timestamp: Date;
    sender: "user" | "ai";
    status?: "pending" | "done";
    actions?: MessageAction[];
  }

  export interface MessageAction {
    id: string;
    label: string;
    variant?: "primary" | "secondary";
    onClick: () => void;
  }

  export interface ContextAttachment {
    id: string;
    type: "goal" | "project";
    title: string;
    url?: string;
  }

  export interface ContextAction {
    id: string;
    icon: React.ReactNode;
    label: string;
    description: string;
  }

  export interface Conversation {
    id: string;
    title: string;
    messages: Message[];
    createdAt: Date;
    updatedAt: Date;
    context?: ContextAttachment;
  }

  export interface Props {
    /**
     * Whether the conversations panel is open
     */
    isOpen: boolean;

    /**
     * Called when the panel should close
     */
    onClose: () => void;

    /**
     * Called when a new message is sent
     */
    onSendMessage?: (message: string, conversationId?: string) => Promise<void>;

    /**
     * List of previous conversations
     */
    conversations?: Conversation[];

    /**
     * Currently active conversation ID
     */
    activeConversationId?: string;

    /**
     * Called when a conversation is selected
     */
    onSelectConversation?: (conversationId: string | undefined) => void;

    /**
     * Called when a new conversation is created
     */
    onCreateConversation?: (action: ContextAction | null) => void;

    /**
     * Called when a conversation title is updated
     */
    onUpdateConversationTitle?: (
      conversationId: string,
      newTitle: string
    ) => void;

    /**
     * Context-aware actions available for current page
     */
    contextActions?: ContextAction[];

    /**
     * Current context attachment (goal, project, etc.)
     */
    contextAttachment?: ContextAttachment;

    /**
     * Initial width of the panel in pixels
     */
    initialWidth?: number;

    /**
     * Minimum width of the panel in pixels
     */
    minWidth?: number;

    /**
     * Maximum width of the panel in pixels
     */
    maxWidth?: number;
  }
}

// Small, unobtrusive copy button for AI messages
function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      // Reset after 2 seconds
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 p-1 rounded hover:bg-surface-dimmed transition-colors opacity-60 hover:opacity-100"
      title={copied ? "Copied!" : "Copy message"}
      aria-label={copied ? "Copied!" : "Copy message"}
    >
      {copied ? (
        <CheckIcon size={14} className="text-green-600" />
      ) : (
        <CopyIcon size={14} className="text-content-dimmed" />
      )}
    </button>
  );
}

export function Conversations({
  isOpen,
  onClose,
  onSendMessage,
  conversations = [],
  activeConversationId,
  onSelectConversation,
  onCreateConversation,
  contextActions = [],
  contextAttachment,
  initialWidth = 448, // (w-md equivalent)
  minWidth = 320, // Minimum usable width
  maxWidth = 600, // Maximum width
}: Conversations.Props) {
  const [mounted, setMounted] = useState(false);
  const [inputMessage, setInputMessage] = useState("");
  const [showConversationsList, setShowConversationsList] = useState(false);
  const [panelWidth, setPanelWidth] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  // Get the active conversation
  const activeConversation = conversations.find(
    (c) => c.id === activeConversationId
  );

  // Handle mounting
  useEffect(() => {
    setMounted(true);
  }, []);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [activeConversation?.messages]);

  // Handle ESC key to close
  useEffect(() => {
    if (isOpen) {
      const handleEsc = (event: KeyboardEvent) => {
        if (event.key === "Escape" && !event.defaultPrevented) {
          onClose();
        }
      };

      document.addEventListener("keydown", handleEsc);
      return () => document.removeEventListener("keydown", handleEsc);
    }
    return undefined;
  }, [isOpen, onClose]);

  // Handle resize functionality
  useEffect(() => {
    if (!isResizing) return;

    // Prevent text selection during resize
    document.body.style.userSelect = "none";
    document.body.style.cursor = "col-resize";

    const handleMouseMove = (e: MouseEvent) => {
      if (!panelRef.current) return;

      const newWidth = window.innerWidth - e.clientX;
      const clampedWidth = Math.max(minWidth, Math.min(maxWidth, newWidth));
      setPanelWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };
  }, [isResizing, minWidth, maxWidth]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  };

  const handleSendMessage = async () => {
    if (!inputMessage.trim() || !onSendMessage) return;

    try {
      await onSendMessage(inputMessage, activeConversationId);
      setInputMessage("");
    } catch (error) {
      console.error("Failed to send message:", error);
    }
  };

  const handleContextAction = async (action: Conversations.ContextAction) => {
    if (!onCreateConversation) return;
    onCreateConversation(action);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const groupConversationsByTime = (
    conversations: Conversations.Conversation[]
  ) => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const weekStart = new Date(today);
    weekStart.setDate(today.getDate() - today.getDay()); // Start of this week (Sunday)

    const groups = {
      Today: [] as Conversations.Conversation[],
      "This Week": [] as Conversations.Conversation[],
      Earlier: [] as Conversations.Conversation[],
    };

    conversations.forEach((conv) => {
      const convDate = new Date(conv.updatedAt);

      if (convDate >= today) {
        groups["Today"].push(conv);
      } else if (convDate >= weekStart) {
        groups["This Week"].push(conv);
      } else {
        groups["Earlier"].push(conv);
      }
    });

    // Filter out empty groups
    return Object.entries(groups).filter(([_, convs]) => convs.length > 0);
  };

  const formatTime = (date: Date | string | number | null | undefined) => {
    // Handle invalid or missing dates gracefully
    if (!date) {
      return "No date";
    }

    let dateObj: Date;

    // Convert to Date object if it's not already
    if (date instanceof Date) {
      dateObj = date;
    } else {
      dateObj = new Date(date);
    }

    // Check if the date is valid
    if (isNaN(dateObj.getTime())) {
      return "Invalid date";
    }

    return new Intl.DateTimeFormat("en-US", {
      hour: "2-digit",
      minute: "2-digit",
      hour12: true,
    }).format(dateObj);
  };

  if (!mounted || !isOpen) {
    return null;
  }

  const panelContent = (
    <div
      ref={panelRef}
      className="fixed inset-y-0 right-0 z-[999] bg-white border-l border-surface-outline shadow-xl flex flex-col"
      style={{ width: `${panelWidth}px` }}
    >
      {/* Resize Handle */}
      <div
        className={`absolute left-0 top-0 bottom-0 w-1 cursor-col-resize transition-all ${
          isResizing
            ? "bg-accent-base"
            : "hover:bg-accent-base/50 bg-surface-outline/50"
        }`}
        onMouseDown={handleResizeStart}
        title="Drag to resize"
      >
        {/* Visual indicator */}
        <div className="absolute left-1 top-1/2 transform -translate-y-1/2 w-3 h-8 flex flex-col justify-center gap-1 pointer-events-none">
          <div className="w-0.5 h-0.5 bg-content-dimmed rounded-full" />
          <div className="w-0.5 h-0.5 bg-content-dimmed rounded-full" />
          <div className="w-0.5 h-0.5 bg-content-dimmed rounded-full" />
        </div>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-surface-outline bg-surface-base">
        <div className="flex items-center gap-2 flex-1 min-w-0 text-sm">
          <h2 className=" text-content-accent">AI Assistant</h2>
        </div>
        <div className="flex items-center gap-1">
          {/* Conversations List Toggle */}
          <button
            onClick={() => setShowConversationsList(!showConversationsList)}
            className="p-2 text-content-dimmed hover:text-content-base hover:bg-surface-highlight rounded transition-colors"
            title="View past conversations"
          >
            <ListIcon size={16} />
          </button>

          {/* New Conversation */}
          <button
            onClick={() => onSelectConversation?.(undefined)}
            className="p-2 text-content-dimmed hover:text-content-base hover:bg-surface-highlight rounded transition-colors"
            title="New conversation"
          >
            <PlusIcon size={16} />
          </button>

          {/* Close */}
          <button
            onClick={onClose}
            className="p-2 text-content-dimmed hover:text-content-base hover:bg-surface-highlight rounded transition-colors"
            title="Close"
          >
            <XIcon size={16} />
          </button>
        </div>
      </div>

      {/* Conversations List Overlay */}
      {showConversationsList && (
        <div className="absolute inset-0 z-10 bg-surface-base border-l border-surface-outline">
          <div className="flex items-center justify-between px-4 py-3 border-b border-surface-outline">
            <h3 className="font-semibold text-content-accent">Conversations</h3>
            <button
              onClick={() => setShowConversationsList(false)}
              className="p-1 text-content-subtle hover:text-content-base hover:bg-surface-highlight rounded transition-colors"
            >
              <XIcon size={16} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto">
            {conversations.length === 0 ? (
              <div className="p-4 text-center text-content-dimmed">
                <FileIcon size={32} className="mx-auto mb-2 opacity-50" />
                <p>No conversations yet</p>
                <p className="text-sm">Start a new conversation to begin</p>
              </div>
            ) : (
              <div className="py-2">
                {groupConversationsByTime(conversations).map(
                  ([groupName, groupConversations]) => (
                    <div key={groupName} className="mb-4">
                      <div className="px-4 py-1 text-xs font-medium text-content-dimmed uppercase tracking-wide">
                        {groupName}
                      </div>
                      <div className="space-y-0.5">
                        {groupConversations.map((conversation) => (
                          <button
                            key={conversation.id}
                            onClick={() => {
                              onSelectConversation?.(conversation.id);
                              setShowConversationsList(false);
                            }}
                            className={`w-full text-left px-4 py-2 hover:bg-surface-highlight transition-colors ${
                              activeConversationId === conversation.id
                                ? "bg-surface-highlight"
                                : ""
                            }`}
                          >
                            <div className="text-sm text-content-base truncate">
                              {conversation.title}
                            </div>
                          </button>
                        ))}
                      </div>
                    </div>
                  )
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Context Attachment */}
      {(activeConversation?.context || contextAttachment) && (
        <div className="px-4 py-3 border-b border-surface-outline bg-surface-base">
          <div className="text-xs text-content-dimmed mb-2 uppercase tracking-wide font-medium">
            Context
          </div>
          <div className="flex items-center gap-2">
            <PaperclipIcon
              size={14}
              className="text-content-base flex-shrink-0"
            />
            <div className="min-w-0 flex-1">
              <div className="text-sm text-content-base truncate">
                {(activeConversation?.context || contextAttachment)?.title}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Context Actions */}
      {contextActions.length > 0 && !activeConversation && (
        <div className="px-4 py-4 bg-surface-base">
          <div className="text-xs text-content-dimmed mb-3 uppercase tracking-wide font-medium">
            Available Actions
          </div>
          <div className="flex flex-col gap-2">
            {contextActions.map((action) => (
              <button
                key={action.id}
                onClick={() => handleContextAction(action)}
                className="px-3 py-2 rounded text-sm font-medium transition-colors bg-surface-highlight text-content-base hover:text-white-1 hover:border-black border border-surface-outline text-left flex items-center gap-3"
              >
                {action.icon}
                <div className="">
                  <span>{action.label}</span>
                  <div className="text-xs text-stone-500">
                    {action.description}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {!activeConversation ? (
          <div className="flex flex-col items-center justify-center h-full text-center text-content-dimmed">
            {contextActions.length === 0 && (
              <>
                <div className="w-16 h-16 bg-accent-base rounded-full flex items-center justify-center mb-4">
                  <BotIcon size={32} className="text-white" />
                </div>
                <h3 className="font-medium mb-2">Welcome to Alfred</h3>
                <p className="text-sm mb-4">
                  Start a conversation to get AI assistance with your work.
                </p>
                <button
                  onClick={() => onCreateConversation?.(null)}
                  className="px-4 py-2 bg-accent-base text-white rounded hover:bg-accent-hover transition-colors"
                >
                  Start New Conversation
                </button>
              </>
            )}
          </div>
        ) : (
          <>
            {activeConversation.messages
              .filter((message) => message.status !== "pending")
              .map((message) => (
                <div
                  key={message.id}
                  className={`flex gap-3 ${
                    message.sender === "user" ? "flex-row-reverse" : ""
                  }`}
                >
                  {/* Avatar */}
                  <div className="flex-shrink-0">
                    {message.sender === "ai" ? (
                      <div className="w-8 h-8 bg-accent-base rounded-full flex items-center justify-center">
                        <BotIcon size={16} className="text-white" />
                      </div>
                    ) : (
                      <BotIcon size={32} className="text-content-dimmed" />
                    )}
                  </div>

                  <div className={`max-w-[75%] space-y-2`}>
                    <div
                      className={`rounded-lg px-3 py-2 relative ${
                        message.sender === "user"
                          ? "bg-callout-info-content text-callout-info-bg shadow-sm"
                          : "bg-surface-highlight text-content-base"
                      }`}
                    >
                      <div className="text-sm whitespace-pre-wrap">
                        {message.content}
                      </div>
                      <div
                        className={`text-xs mt-1 ${
                          message.sender === "user"
                            ? "text-white/70"
                            : "text-content-dimmed"
                        }`}
                      >
                        {formatTime(message.timestamp)}
                      </div>

                      {/* Copy button for AI messages */}
                      {message.sender === "ai" && (
                        <CopyButton text={message.content} />
                      )}
                    </div>

                    {/* Message Actions */}
                    {message.actions && message.actions.length > 0 && (
                      <div className="flex flex-wrap gap-2">
                        {message.actions.map((action) => (
                          <button
                            key={action.id}
                            onClick={action.onClick}
                            className="px-3 py-2 rounded text-sm font-medium transition-colors bg-surface-highlight text-content-base hover:text-white-1 hover:bg-brand-1 border border-surface-outline text-left"
                          >
                            {action.label}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}

            {/* Typing indicator for pending AI messages */}
            {activeConversation.messages.some(
              (message) =>
                message.sender === "ai" && message.status === "pending"
            ) && (
              <div className="flex gap-3">
                {/* AI Avatar */}
                <div className="flex-shrink-0">
                  <div className="w-8 h-8 bg-accent-base rounded-full flex items-center justify-center">
                    <BotIcon size={16} className="text-white" />
                  </div>
                </div>

                <div className="max-w-[75%]">
                  <div className="rounded-lg px-3 py-2 bg-surface-highlight text-content-base">
                    <div className="flex items-center gap-2">
                      <BounceLoader size={16} color="#6B7280" />
                      <span className="text-sm text-content-dimmed">
                        Alfred is typing...
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </>
        )}
      </div>

      {/* Input Area */}
      {activeConversation && (
        <div className="border-t border-surface-outline p-4">
          <div className="flex gap-2">
            <textarea
              ref={inputRef}
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder="Type your message..."
              rows={1}
              className="flex-1 resize-none border border-surface-outline rounded px-3 py-2 text-sm bg-surface-base text-content-base placeholder-content-dimmed focus:outline-none focus:ring-2 focus:ring-accent-base focus:border-transparent"
              style={{
                minHeight: "38px",
                maxHeight: "120px",
              }}
            />
            <Button onClick={handleSendMessage} disabled={!inputMessage.trim()}>
              Send
            </Button>
          </div>
        </div>
      )}
    </div>
  );

  return createPortal(panelContent, document.body);
}

function BounceLoader({
  size = 16,
  color = "#000",
}: {
  size?: number;
  color?: string;
}) {
  return (
    <div
      className="flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <div
        className="w-2 h-2 bg-current rounded-full animate-bounce"
        style={{ backgroundColor: color, animationDelay: "0s" }}
      ></div>
      <div
        className="w-2 h-2 bg-current rounded-full animate-bounce mx-1"
        style={{ backgroundColor: color, animationDelay: "0.2s" }}
      ></div>
      <div
        className="w-2 h-2 bg-current rounded-full animate-bounce"
        style={{ backgroundColor: color, animationDelay: "0.4s" }}
      ></div>
    </div>
  );
}

export default Conversations;

// Re-export the hook and types
export { useConversations } from "./useConversations";
export type {
  UseConversationsOptions,
  UseConversationsReturn,
} from "./useConversations";
