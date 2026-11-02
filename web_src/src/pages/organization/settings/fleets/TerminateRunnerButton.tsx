import { useEffect, useRef, useState } from "react";
import { organizationsTerminateFleetRunner, type SuperplaneOrganizationsRunner } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useRefreshOrganizationFleets } from "@/hooks/useOrganizationFleetManagement";

export function TerminateRunnerButton({
  organizationId,
  fleetId,
  runner: record,
}: {
  organizationId: string;
  fleetId: string;
  runner: SuperplaneOrganizationsRunner;
}) {
  const runnerId = record.id!;
  const refresh = useRefreshOrganizationFleets(organizationId);
  const [confirming, setConfirming] = useState(false);
  const [requested, setRequested] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const terminate = async () => {
    if (submitting || requested || !runnerId) return;
    const request = new AbortController();
    abort.current = request;
    setSubmitting(true);
    setError("");
    try {
      await organizationsTerminateFleetRunner(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId, runnerId },
          query: { fleetId, states: record.state ? [record.state] : [] },
          signal: request.signal,
        }),
      );
      if (request.signal.aborted) return;
      setRequested(true);
      setConfirming(false);
    } catch {
      if (request.signal.aborted) return;
      setError("Could not terminate the runner. If it is busy and reusable, wait for the task to finish and retry.");
      setConfirming(false);
    } finally {
      if (!request.signal.aborted) {
        setSubmitting(false);
        refresh();
      }
    }
  };
  return (
    <div className="space-y-2">
      {requested && record.state !== "terminated" && (
        <p role="status" className="text-[12px] text-muted-foreground">
          Termination requested.
        </p>
      )}
      {error && <p role="alert">{error}</p>}
      {record.state !== "terminated" && (
        <Button
          size="xs"
          variant="outline"
          aria-label="Terminate runner"
          disabled={requested || submitting}
          onClick={() => setConfirming(true)}
        >
          Terminate
        </Button>
      )}
      {confirming && (
        <Dialog open onOpenChange={setConfirming}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Terminate runner?</DialogTitle>
              <DialogDescription>Termination does not delete the host machine.</DialogDescription>
            </DialogHeader>
            {record.state === "busy" && record.ephemeral && (
              <p>This request cancels active work. Termination completes asynchronously.</p>
            )}
            {record.state === "busy" && !record.ephemeral && (
              <p>A busy reusable runner cannot be terminated. Wait for its task to finish and retry.</p>
            )}
            <Button variant="destructive" disabled={submitting} onClick={() => void terminate()}>
              Confirm termination
            </Button>
            <Button variant="outline" onClick={() => setConfirming(false)}>
              Keep runner
            </Button>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
