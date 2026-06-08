import { Eye, Rocket, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { getResponseErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Sentry } from "@/sentry";

export interface DraftActionsWidgetProps {
  versionId: string;
  message?: string;
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  onDismiss?: () => void;
}

type DraftAction = "publish" | "discard";

// Server messages indicating the draft this widget references is no longer
// actionable (already published, deleted, or owned by someone else). Hitting
// these is an expected race when the user has multiple tabs / CLIs open, so
// we silently dismiss instead of surfacing an error.
const STALE_DRAFT_MARKERS = [
  "only draft versions can be published",
  "only draft versions can be discarded",
  "version not found",
  "version owner mismatch",
  "version is not your editable draft",
  "version is not a registered draft branch",
];

const ACTION_LABELS: Record<DraftAction, { progress: string; idle: string; success: string; failure: string }> = {
  publish: {
    progress: "Publishing...",
    idle: "Publish",
    success: "Draft published.",
    failure: "Failed to publish draft.",
  },
  discard: {
    progress: "Discarding...",
    idle: "Discard",
    success: "Draft discarded.",
    failure: "Failed to discard draft.",
  },
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
    const labels = ACTION_LABELS[action];
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
        showSuccessToast(labels.success);
        onDismiss?.();
        return;
      }

      const apiMessage = await getResponseErrorMessage(response, labels.failure);

      // 4xx are expected business-logic outcomes (most commonly the draft was
      // already published / discarded from another tab or the CLI). Surface
      // them as a normal toast and silently dismiss the widget when the
      // server tells us the draft is no longer there. Do NOT log to
      // console.error / Sentry — captureConsoleIntegration would otherwise
      // promote these to noisy error events.
      if (response.status >= 400 && response.status < 500) {
        if (isStaleDraftMessage(apiMessage)) {
          onDismiss?.();
          return;
        }
        showErrorToast(apiMessage);
        return;
      }

      // 5xx (and anything else): a real server-side bug. Show the user a
      // toast and report a structured Sentry event with enough context to
      // debug, instead of relying on captureConsoleIntegration to scrape an
      // unstructured `console.error` string.
      showErrorToast(apiMessage);
      reportDraftActionFailure({
        action,
        canvasId,
        versionId,
        status: response.status,
        message: apiMessage,
      });
    } catch (err) {
      // Network / fetch failure (offline, CORS, aborted, etc.). Show the
      // user a toast and capture the original Error so Sentry has a stack.
      showErrorToast(labels.failure);
      reportDraftActionFailure({
        action,
        canvasId,
        versionId,
        status: 0,
        message: err instanceof Error ? err.message : String(err),
        cause: err,
      });
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
        {busy === "discard" ? ACTION_LABELS.discard.progress : ACTION_LABELS.discard.idle}
      </Button>
      <Button
        variant="default"
        size="sm"
        onClick={handlePublish}
        disabled={busy !== null}
        className="text-xs h-7 gap-1"
      >
        <Rocket size={12} />
        {busy === "publish" ? ACTION_LABELS.publish.progress : ACTION_LABELS.publish.idle}
      </Button>
    </div>
  );
}

function isStaleDraftMessage(message: string): boolean {
  const lower = message.toLowerCase();
  return STALE_DRAFT_MARKERS.some((marker) => lower.includes(marker));
}

interface DraftActionFailure {
  action: DraftAction;
  canvasId: string;
  versionId: string;
  status: number;
  message: string;
  cause?: unknown;
}

function reportDraftActionFailure({ action, canvasId, versionId, status, message, cause }: DraftActionFailure): void {
  const error = cause instanceof Error ? cause : new Error(`Draft ${action} failed: ${status} ${message}`);

  Sentry.withScope((scope) => {
    scope.setTag("draft.action", action);
    scope.setTag("draft.status", String(status));
    scope.setExtra("canvasId", canvasId);
    scope.setExtra("versionId", versionId);
    scope.setExtra("apiMessage", message);
    Sentry.captureException(error);
  });
}
