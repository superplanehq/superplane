import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { AlertTriangle } from "lucide-react";
import { useState } from "react";

type ResetBacklogResult = {
  reset: number;
  failures: { canvas_id: string; name: string; error: string }[];
};

export function OrgResetBacklogDefaults({ orgId }: { orgId: string }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/backlog-defaults/reset`, {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error("reset failed");
      }
      const result = (await response.json()) as ResetBacklogResult;
      setConfirmOpen(false);
      setMessage(resetBacklogResultMessage(result));
    } catch {
      setError("Could not reset Backlog automations.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mb-4">
      <Button type="button" variant="outline" size="sm" onClick={() => setConfirmOpen(true)} disabled={busy}>
        Reset Backlog defaults
      </Button>
      {message ? <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">{message}</Text> : null}
      {error ? <Text className="mt-2 text-sm text-red-600 dark:text-red-400">{error}</Text> : null}

      <Dialog open={confirmOpen} onClose={() => (busy ? undefined : setConfirmOpen(false))} size="md">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 rounded-full bg-red-100 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <AlertTriangle size={20} />
          </div>
          <DialogTitle className="text-gray-800 dark:text-gray-100">Reset Backlog automations</DialogTitle>
        </div>
        <DialogDescription className="text-sm text-gray-600 mt-2 space-y-2 dark:text-gray-400">
          <p>
            This action replaces every Backlog automation in this organization with the current SuperPlane defaults.
          </p>
          <p>Custom prompts and graph changes in those automations are lost. Other automations stay the same.</p>
          <p>You cannot undo this action.</p>
        </DialogDescription>
        <DialogActions>
          <Button variant="destructive" onClick={() => void handleConfirm()} disabled={busy}>
            {busy ? "Resetting..." : "Reset Backlog automations"}
          </Button>
          <Button variant="outline" onClick={() => setConfirmOpen(false)} disabled={busy}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}

export function resetBacklogResultMessage(result: ResetBacklogResult): string {
  const failed = result.failures?.length ?? 0;
  if (result.reset === 0 && failed === 0) {
    return "No Backlog automations to reset.";
  }
  const resetText = result.reset === 1 ? "Reset 1 Backlog automation." : `Reset ${result.reset} Backlog automations.`;
  if (failed === 0) {
    return resetText;
  }
  const failText = failed === 1 ? "1 failed." : `${failed} failed.`;
  return `${resetText} ${failText}`;
}
