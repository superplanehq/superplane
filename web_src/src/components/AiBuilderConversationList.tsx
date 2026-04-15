import { useState } from "react";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import type { AiChatSession } from "@/ui/BuildingBlocksSidebar/agentChat";
import { cn } from "@/lib/utils";
import { ArrowLeft, Trash2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

export type ConversationListProps = {
  chatSessions: AiChatSession[];
  currentChatId: string | null;
  isLoadingChatSessions: boolean;
  isGeneratingResponse: boolean;
  onSelectChat: (chatId: string) => void;
  onDeleteChat?: (chatId: string) => void;
  onStartNewSession: () => void;
  title?: string;
  className?: string;
  fillAvailable?: boolean;
};

function formatSessionDate(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function DeleteConfirmDialog({
  pendingDeleteId,
  onClose,
  onConfirm,
}: {
  pendingDeleteId: string | null;
  onClose: () => void;
  onConfirm: (id: string) => void;
}) {
  return (
    <AlertDialog open={pendingDeleteId !== null} onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete conversation?</AlertDialogTitle>
          <AlertDialogDescription>
            This will permanently delete this conversation and all its messages. This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-white hover:bg-destructive/90"
            onClick={() => {
              if (pendingDeleteId) {
                onConfirm(pendingDeleteId);
              }
              onClose();
            }}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function CurrentSessionHeader({ session, isLoading }: { session: AiChatSession | undefined; isLoading: boolean }) {
  if (isLoading) {
    return <span className="text-xs text-slate-500">Loading…</span>;
  }

  if (session) {
    return (
      <div
        className="flex min-w-0 flex-1 items-center justify-between gap-2"
        title={session.createdAt ? formatSessionDate(session.createdAt) : undefined}
      >
        <div className="min-w-0 truncate text-sm font-medium text-slate-800">{session.title}</div>
        {session.createdAt ? (
          <TimeAgo date={session.createdAt} className="shrink-0 text-[11px] tabular-nums text-slate-500" />
        ) : null}
      </div>
    );
  }

  return <span className="text-sm text-slate-600">Conversation</span>;
}

export function ConversationList({
  chatSessions,
  currentChatId,
  isLoadingChatSessions,
  isGeneratingResponse,
  onSelectChat,
  onDeleteChat,
  onStartNewSession,
  title,
  className,
  fillAvailable = false,
}: ConversationListProps) {
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);

  const currentSession = currentChatId ? chatSessions.find((s) => s.id === currentChatId) : undefined;

  return (
    <div
      className={cn("border-b border-border px-2 py-2 space-y-2", fillAvailable && "flex min-h-0 flex-col", className)}
    >
      <div className="flex min-w-0 items-center gap-2">
        {currentChatId ? (
          <>
            <Button
              size="icon-xs"
              variant="ghost"
              onClick={onStartNewSession}
              disabled={isGeneratingResponse}
              aria-label="Back to new chat"
              title="Back"
              className="shrink-0"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <CurrentSessionHeader session={currentSession} isLoading={isLoadingChatSessions} />
          </>
        ) : (
          <p className="text-[11px] font-medium uppercase tracking-[0.08em] text-slate-500">
            {title ?? "Conversations"}
          </p>
        )}
      </div>

      {!currentChatId ? (
        <div
          className={cn(
            fillAvailable ? "min-h-0 flex-1 overflow-y-auto" : "max-h-28 overflow-y-auto",
            "bg-transparent",
            fillAvailable ? "space-y-2" : "space-y-1",
            "[scrollbar-width:thin] [scrollbar-color:rgb(203_213_225)_transparent]",
            "[&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-slate-300/70 [&::-webkit-scrollbar-track]:bg-transparent",
          )}
        >
          {isLoadingChatSessions ? (
            <div className="text-xs text-gray-500 px-2 py-2">Loading conversations...</div>
          ) : null}
          {!isLoadingChatSessions && chatSessions.length === 0 ? (
            <div className="text-xs text-gray-500 px-2 py-2">No conversations yet.</div>
          ) : null}

          {chatSessions.map((session) => {
            return (
              <div key={session.id} className="group relative">
                <button
                  type="button"
                  onClick={() => onSelectChat(session.id)}
                  disabled={isGeneratingResponse}
                  title={session.createdAt ? formatSessionDate(session.createdAt) : undefined}
                  className="w-full rounded-md border border-slate-200 bg-white px-2 py-2 text-left text-slate-700 transition-colors hover:bg-slate-50"
                >
                  <div className="flex min-w-0 items-center justify-between gap-2">
                    <div className="min-w-0 truncate text-sm font-medium">{session.title}</div>
                    <div className="flex shrink-0 items-center gap-1">
                      {session.createdAt ? (
                        <TimeAgo
                          date={session.createdAt}
                          className="shrink-0 text-[11px] tabular-nums text-slate-500 group-hover:hidden"
                        />
                      ) : null}
                    </div>
                  </div>
                </button>
                {onDeleteChat ? (
                  <button
                    type="button"
                    onClick={(event) => {
                      event.stopPropagation();
                      setPendingDeleteId(session.id);
                    }}
                    disabled={isGeneratingResponse}
                    className="absolute top-1/2 right-2 hidden -translate-y-1/2 rounded p-1 text-slate-400 transition-colors hover:bg-slate-100 hover:text-red-500 group-hover:block"
                    aria-label="Delete conversation"
                    title="Delete"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                ) : null}
              </div>
            );
          })}
        </div>
      ) : null}

      {onDeleteChat ? (
        <DeleteConfirmDialog
          pendingDeleteId={pendingDeleteId}
          onClose={() => setPendingDeleteId(null)}
          onConfirm={onDeleteChat}
        />
      ) : null}
    </div>
  );
}
