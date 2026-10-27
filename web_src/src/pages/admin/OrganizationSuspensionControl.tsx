import { useState } from "react";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { OrganizationOverview } from "./OrgOverviewPanel";

interface SuspensionEvent {
  id: string;
  action: string;
  account_id: string;
  reason: string;
  created_at: string;
  cancellation_completed_at?: string;
}

function useSuspensionActions({
  organization,
  onChange,
}: {
  organization: OrganizationOverview;
  onChange: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [history, setHistory] = useState<SuspensionEvent[] | null>(null);
  const suspended = Boolean(organization.suspended_at);
  const action = suspended ? "Reactivate organization" : "Suspend organization";

  async function changeSuspension() {
    setPending(true);
    setError(null);
    try {
      const response = await fetch(
        `/admin/api/organizations/${organization.id}/${suspended ? "reactivate" : "suspend"}`,
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(suspended ? {} : { reason: reason.trim() }),
        },
      );
      if (!response.ok) {
        setError((await response.text()).trim() || "Could not change organization suspension.");
        return;
      }
      setOpen(false);
      setReason("");
      setHistory(null);
      onChange();
    } catch {
      setError("Could not change organization suspension.");
    } finally {
      setPending(false);
    }
  }

  async function loadHistory() {
    setError(null);
    try {
      const response = await fetch(`/admin/api/organizations/${organization.id}/suspension-history`, {
        credentials: "include",
      });
      if (!response.ok) throw new Error();
      setHistory(await response.json());
    } catch {
      setError("Could not load suspension history.");
    }
  }

  return {
    open,
    setOpen,
    reason,
    setReason,
    pending,
    error,
    setError,
    history,
    suspended,
    action,
    changeSuspension,
    loadHistory,
  };
}

export function OrganizationSuspensionControl({
  organization,
  onChange,
}: {
  organization: OrganizationOverview;
  onChange: () => void;
}) {
  const actions = useSuspensionActions({ organization, onChange });
  const { setOpen, suspended, action, error, setError, history, loadHistory } = actions;
  return (
    <section aria-label="Organization suspension" className="mb-4 rounded-md border border-border p-4">
      <h2 className="font-medium">{suspended ? "Organization suspended" : "Organization active"}</h2>
      {suspended ? (
        <p className="mt-1 text-sm text-muted-foreground">
          {organization.suspension_reason} ·{" "}
          {organization.suspended_at ? new Date(organization.suspended_at).toLocaleString() : ""}
          {organization.suspended_by_account_id ? ` · Admin: ${organization.suspended_by_account_id}` : ""}
        </p>
      ) : null}
      <div className="mt-3 flex gap-2">
        <Button
          variant={suspended ? "outline" : "destructive"}
          onClick={() => {
            setError(null);
            setOpen(true);
          }}
        >
          {action}
        </Button>
        <Button variant="outline" onClick={() => void loadHistory()}>
          View suspension history
        </Button>
      </div>
      {error ? (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}
      <SuspensionHistory history={history} />
      <SuspensionDialog actions={actions} />
    </section>
  );
}

function SuspensionHistory({ history }: { history: SuspensionEvent[] | null }) {
  return (
    <>
      {history ? (
        <ul className="mt-3 space-y-2 text-sm">
          {history.length === 0 ? (
            <li>No suspension history.</li>
          ) : (
            history.map((event) => (
              <li key={event.id}>
                {event.action === "suspend" ? "Suspended" : "Reactivated"} ·{" "}
                {new Date(event.created_at).toLocaleString()} · Admin: {event.account_id}
                {event.reason ? ` · ${event.reason}` : ""}
                {event.action === "suspend" && !event.cancellation_completed_at ? " · Work is stopping." : ""}
              </li>
            ))
          )}
        </ul>
      ) : null}
    </>
  );
}

function SuspensionDialog({ actions }: { actions: ReturnType<typeof useSuspensionActions> }) {
  const { open, setOpen, reason, setReason, pending, error, suspended, action, changeSuspension } = actions;
  return (
    <Dialog
      open={open}
      onClose={() => {
        if (!pending) setOpen(false);
      }}
      size="md"
      role="dialog"
      aria-modal="true"
      aria-label={action}
    >
      <DialogTitle>{action}?</DialogTitle>
      <DialogDescription>
        {suspended
          ? "Future activity can start again. Canceled work and missed schedules will not restart."
          : "Members keep read-only access. New work stops, and SuperPlane requests cancellation of active work."}
      </DialogDescription>
      {!suspended ? (
        <div className="my-3">
          <Label htmlFor="organization-suspension-reason">Suspension reason</Label>
          <Textarea
            id="organization-suspension-reason"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={2000}
            disabled={pending}
          />
          <p className="mt-1 text-xs text-muted-foreground">Only installation admins can view this reason.</p>
        </div>
      ) : null}
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
      <DialogActions>
        <Button variant="outline" disabled={pending} onClick={() => setOpen(false)}>
          Cancel
        </Button>
        <Button
          variant={suspended ? "default" : "destructive"}
          disabled={pending || (!suspended && !reason.trim())}
          onClick={() => void changeSuspension()}
        >
          {pending ? "Saving..." : action}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
