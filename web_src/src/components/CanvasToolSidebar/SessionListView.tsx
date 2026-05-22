import { ArrowLeft, Circle } from "lucide-react";
import type { CanvasSession } from "@/hooks/useAgentChats";

interface SessionListViewProps {
  sessions: CanvasSession[];
  currentUserId: string;
  onSelectSession: (sessionId: string) => void;
  onBack: () => void;
}

export function SessionListView({ sessions, currentUserId, onSelectSession, onBack }: SessionListViewProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center gap-2 border-b border-border px-3 py-2">
        <button type="button" onClick={onBack} className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-700">
          <ArrowLeft className="size-3" />
          Back to my session
        </button>
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="px-3 py-2">
          <p className="text-xs font-medium text-slate-500 uppercase tracking-wide mb-2">Sessions on this canvas</p>
          {sessions.length === 0 ? (
            <p className="text-sm text-slate-400">No sessions yet.</p>
          ) : (
            <div className="space-y-1">
              {sessions.map((session) => (
                <button
                  key={session.id}
                  type="button"
                  onClick={() => onSelectSession(session.id)}
                  className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left hover:bg-slate-50 transition-colors"
                >
                  {session.userAvatarUrl ? (
                    <img src={session.userAvatarUrl} alt="" className="size-7 rounded-full" />
                  ) : (
                    <div className="flex size-7 items-center justify-center rounded-full bg-slate-200 text-xs font-medium text-slate-600">
                      {(session.userName?.[0] ?? "?").toUpperCase()}
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-slate-800 truncate">
                        {session.userName || "Unknown"}
                      </span>
                      {session.userId === currentUserId && (
                        <span className="text-[10px] text-slate-400">(you)</span>
                      )}
                    </div>
                    <div className="flex items-center gap-1.5 text-[10px] text-slate-400">
                      <Circle
                        className={`size-1.5 fill-current ${session.status === "streaming" ? "text-green-500" : "text-slate-300"}`}
                      />
                      <span>{session.status === "streaming" ? "Active" : formatTimeAgo(session.lastActivityAt)}</span>
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function formatTimeAgo(timestamp: string | null): string {
  if (!timestamp) return "No activity";
  const diff = Date.now() - new Date(timestamp).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
