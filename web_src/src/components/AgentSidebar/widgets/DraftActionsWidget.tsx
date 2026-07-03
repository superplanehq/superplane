import { Eye, Rocket, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

export interface DraftActionsWidgetProps {
  versionId?: string;
  message?: string;
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  onDismiss?: () => void;
}

export function DraftActionsWidget({
  versionId,
  message,
  canvasId,
  organizationId,
  isEditing,
  onDismiss,
}: DraftActionsWidgetProps) {
  const [busy, setBusy] = useState<"commit" | "discard" | null>(null);

  const handleViewInEditor = () => {
    if (!versionId) {
      return;
    }
    window.dispatchEvent(new CustomEvent("agent:view-version", { detail: { versionId } }));
  };

  const sendRequest = (method: string, url: string, body?: string) =>
    fetch(url, {
      method,
      headers: {
        "Content-Type": "application/json",
        "x-organization-id": organizationId,
      },
      credentials: "include",
      ...(body !== undefined ? { body } : {}),
    });

  const runAction = async (action: "commit" | "discard", run: () => Promise<Response>) => {
    setBusy(action);
    try {
      const response = await run();
      if (response.ok) {
        onDismiss?.();
      } else {
        const text = await response.text();
        console.error(`${action} failed:`, response.status, text);
      }
    } catch (err) {
      console.error(`Failed to ${action}:`, err);
    } finally {
      setBusy(null);
    }
  };

  const handleCommit = () =>
    runAction("commit", async () => {
      const commitMessage = message?.trim() || "Apply agent changes";
      return sendRequest("POST", `/api/v1/canvases/${canvasId}/staging/commit`, JSON.stringify({ commitMessage }));
    });

  const handleDiscard = () => runAction("discard", () => sendRequest("DELETE", `/api/v1/canvases/${canvasId}/staging`));

  return (
    <div className="flex items-center gap-2">
      {message && <span className="text-xs text-slate-600 flex-1 truncate">{message}</span>}
      {!message && <span className="text-xs text-slate-600 flex-1">Staging ready</span>}
      {!isEditing && versionId && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleViewInEditor}
          className="text-xs h-7 gap-1"
          disabled={busy !== null}
        >
          <Eye size={12} />
          See changes
        </Button>
      )}
      <Button
        variant="outline"
        size="sm"
        onClick={handleDiscard}
        disabled={busy !== null}
        className="text-xs h-7 gap-1 text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200"
      >
        <Trash2 size={12} />
        {busy === "discard" ? "Discarding..." : "Discard"}
      </Button>
      <Button variant="default" size="sm" onClick={handleCommit} disabled={busy !== null} className="text-xs h-7 gap-1">
        <Rocket size={12} />
        {busy === "commit" ? "Committing..." : "Commit"}
      </Button>
    </div>
  );
}
