import { useEffect, useRef, useState } from "react";
import { organizationsUpdateRunnerFleet, type SuperplaneOrganizationsFleet } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function FleetEnabledButton({
  organizationId,
  fleet,
  refresh,
}: {
  organizationId: string;
  fleet: SuperplaneOrganizationsFleet;
  refresh: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const saveEnabled = async (enabled: boolean) => {
    if (saving || !fleet.id) return;
    const request = new AbortController();
    abort.current = request;
    setSaving(true);
    setError("");
    try {
      await organizationsUpdateRunnerFleet(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId, fleetId: fleet.id },
          body: { enabled },
          signal: request.signal,
        }),
      );
      if (!request.signal.aborted) {
        setConfirming(false);
        refresh();
      }
    } catch {
      if (!request.signal.aborted)
        setError(`Could not ${enabled ? "enable" : "disable"} the fleet. Check access and retry.`);
    } finally {
      if (!request.signal.aborted) setSaving(false);
    }
  };
  return (
    <div className="relative">
      <Button
        size="sm"
        variant="outline"
        onClick={() => (fleet.enabled ? setConfirming(true) : void saveEnabled(true))}
        disabled={saving}
      >
        {saving ? "Saving…" : fleet.enabled ? "Disable fleet" : "Enable fleet"}
      </Button>
      <Dialog
        open={confirming}
        onOpenChange={(open) => {
          if (!saving) setConfirming(open);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable fleet?</DialogTitle>
            <DialogDescription>
              Disabling {fleet.id} stops new work and runner registration. Active work continues.
            </DialogDescription>
          </DialogHeader>
          {error && (
            <p role="alert" className="text-[12px] text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirming(false)} disabled={saving}>
              Keep fleet enabled
            </Button>
            <Button variant="destructive" onClick={() => void saveEnabled(false)} disabled={saving}>
              {saving ? "Disabling…" : "Confirm disable"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {error && !confirming && (
        <p role="alert" className="mt-2 max-w-64 text-[12px] text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
