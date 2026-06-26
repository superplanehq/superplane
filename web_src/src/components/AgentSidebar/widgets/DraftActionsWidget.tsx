import { Eye, Rocket, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

export interface DraftActionsWidgetProps {
  versionId: string;
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
  const [busy, setBusy] = useState<"publish" | "discard" | null>(null);

  const handleViewInEditor = () => {
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

  const runAction = async (action: "publish" | "discard", run: () => Promise<Response>) => {
    setBusy(action);
    try {
      const response = await run();
      if (response.ok) {
        onDismiss?.();
        return;
      }
      // The draft can disappear out from under us (e.g. another tab already
      // discarded it, or the agent's last publish removed it). Treat 404 as
      // an idempotent success so the action bar dismisses without surfacing
      // a Sentry-noisy "version not found" error.
      if (response.status === 404) {
        onDismiss?.();
        return;
      }
      const text = await response.text();
      console.error(`${action} failed:`, response.status, text);
    } catch (err) {
      console.error(`Failed to ${action}:`, err);
    } finally {
      setBusy(null);
    }
  };

  // The agent writes draft edits into workflow_staged_files (the same layer the UI
  // editor stages into), and publish materializes the draft version row only.
  // Commit any pending staging before publishing so the agent's staged edits
  // are included; otherwise publish would ship the last committed version and
  // silently drop them. Commit is a no-op when there is nothing staged.
  const handlePublish = () =>
    runAction("publish", async () => {
      const commitResponse = await sendRequest(
        "POST",
        `/api/v1/canvases/${canvasId}/versions/${versionId}/staging/commit`,
        "{}",
      );
      if (!commitResponse.ok) {
        return commitResponse;
      }
      return sendRequest("PATCH", `/api/v1/canvases/${canvasId}/versions/${versionId}/publish`, "{}");
    });

  const handleDiscard = () =>
    runAction("discard", () => sendRequest("DELETE", `/api/v1/canvases/${canvasId}/versions/${versionId}`));

  return (
    <div className="flex items-center gap-2">
      {message && <span className="text-xs text-slate-600 flex-1 truncate">{message}</span>}
      {!message && <span className="text-xs text-slate-600 flex-1">Draft ready</span>}
      {!isEditing && (
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
      <Button
        variant="default"
        size="sm"
        onClick={handlePublish}
        disabled={busy !== null}
        className="text-xs h-7 gap-1"
      >
        <Rocket size={12} />
        {busy === "publish" ? "Publishing..." : "Publish"}
      </Button>
    </div>
  );
}
