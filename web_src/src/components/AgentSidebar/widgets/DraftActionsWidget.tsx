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

type DraftAction = "publish" | "discard";

// Server messages that indicate the widget is showing a stale draft (it was
// already published / discarded by someone else, or no longer exists). We
// silently dismiss in those cases so the user is not blocked by a stale UI.
const STALE_DRAFT_MARKERS = ["only draft versions can be published", "version not found", "version owner mismatch"];

export function DraftActionsWidget({
  versionId,
  message,
  canvasId,
  organizationId,
  isEditing,
  onDismiss,
}: DraftActionsWidgetProps) {
  const [busy, setBusy] = useState<DraftAction | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleViewInEditor = () => {
    window.dispatchEvent(new CustomEvent("agent:view-version", { detail: { versionId } }));
  };

  const callApi = async (method: string, url: string, action: DraftAction) => {
    setBusy(action);
    setError(null);
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

      const text = await response.text();
      const apiMessage = extractApiMessage(text);

      // 4xx responses are expected business-logic outcomes (e.g. the draft was
      // already published from another tab/CLI). Surface them inline and, when
      // the widget is clearly stale, dismiss it — but never `console.error`,
      // since that ships noise to Sentry via captureConsoleIntegration.
      if (response.status >= 400 && response.status < 500) {
        if (isStaleDraftMessage(apiMessage)) {
          onDismiss?.();
          return;
        }
        setError(apiMessage ?? `Failed to ${action} (HTTP ${response.status}).`);
        return;
      }

      setError(apiMessage ?? `Failed to ${action} (HTTP ${response.status}).`);
      console.error(`${action} failed:`, response.status, text);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action}.`);
      console.error(`Failed to ${action}:`, err);
    } finally {
      setBusy(null);
    }
  };

  const handlePublish = () => callApi("PATCH", `/api/v1/canvases/${canvasId}/versions/${versionId}/publish`, "publish");

  const handleDiscard = () => callApi("DELETE", `/api/v1/canvases/${canvasId}/versions/${versionId}`, "discard");

  return (
    <div className="flex flex-col gap-1">
      {error && (
        <div role="alert" className="text-xs text-red-600">
          {error}
        </div>
      )}
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
    </div>
  );
}

function extractApiMessage(text: string): string | null {
  const trimmed = text.trim();
  if (!trimmed) return null;

  try {
    const parsed = JSON.parse(trimmed) as { message?: unknown };
    if (typeof parsed.message === "string" && parsed.message.trim().length > 0) {
      return parsed.message.trim();
    }
  } catch {
    // Body was not JSON; fall through to the raw text.
  }

  return trimmed;
}

function isStaleDraftMessage(message: string | null): boolean {
  if (!message) return false;
  const lower = message.toLowerCase();
  return STALE_DRAFT_MARKERS.some((marker) => lower.includes(marker));
}
