import { Eye, Rocket, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { getResponseErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

export interface DraftActionsWidgetProps {
  versionId: string;
  message?: string;
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  onDismiss?: () => void;
}

type DraftAction = "publish" | "discard";

const ACTION_FALLBACK_MESSAGE: Record<DraftAction, string> = {
  publish: "Failed to publish draft.",
  discard: "Failed to discard draft.",
};

export function DraftActionsWidget({
  versionId,
  message,
  canvasId,
  organizationId,
  isEditing,
  onDismiss,
}: DraftActionsWidgetProps) {
  const [busy, setBusy] = useState<DraftAction | null>(null);

  const handleViewInEditor = () => {
    window.dispatchEvent(new CustomEvent("agent:view-version", { detail: { versionId } }));
  };

  const callApi = async (method: string, url: string, action: DraftAction) => {
    const fallback = ACTION_FALLBACK_MESSAGE[action];
    setBusy(action);
    try {
      const response = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
          "x-organization-id": organizationId,
        },
        credentials: "include",
      });
      if (response.ok) {
        onDismiss?.();
        return;
      }
      // Surface a user-friendly toast and avoid logging raw response bodies
      // (e.g. HTML 502 error pages) which would otherwise be forwarded to
      // Sentry by the global console.error capture integration.
      const reason = await getResponseErrorMessage(response, fallback);
      showErrorToast(reason);
    } catch {
      showErrorToast(fallback);
    } finally {
      setBusy(null);
    }
  };

  const handlePublish = () => callApi("PATCH", `/api/v1/canvases/${canvasId}/versions/${versionId}/publish`, "publish");

  const handleDiscard = () => callApi("DELETE", `/api/v1/canvases/${canvasId}/versions/${versionId}`, "discard");

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
          See in Editor
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
