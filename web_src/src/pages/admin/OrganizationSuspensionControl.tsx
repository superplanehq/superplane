import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  adminOrganizationsListSuspensionHistory,
  adminOrganizationsSuspendOrganization,
  adminOrganizationsReactivateOrganization,
} from "@/api-client/sdk.gen";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { organizationKeys } from "@/hooks/useOrganizationData";
import type { OrganizationsSuspensionResponse } from "@/api-client";

function cancellationPending(status?: OrganizationsSuspensionResponse) {
  return Boolean(
    status?.suspended && status.events?.some((event) => event.action === "suspend" && !event.cancellationCompletedAt),
  );
}

function useSuspensionHistory(id: string) {
  return useQuery({
    queryKey: ["admin-suspension", id],
    queryFn: async () =>
      (await adminOrganizationsListSuspensionHistory({ path: { id }, credentials: "include", throwOnError: true }))
        .data,
    refetchInterval: (query) => (cancellationPending(query.state.data) ? 5000 : false),
  });
}

function SuspensionHistory({ status }: { status?: OrganizationsSuspensionResponse }) {
  return (
    <>
      <h3 className="font-semibold">Suspension history</h3>
      <ul>
        {status?.events?.map((event) => (
          <li key={event.id} className="py-2">
            <p>
              {event.action === "suspend" ? "Suspended" : "Reactivated"} —{" "}
              {event.createdAt ? new Date(event.createdAt).toLocaleString() : ""}
            </p>
            <p>Actor: {event.accountId}</p>
            {event.reason ? <p>{event.reason}</p> : null}
            {event.action === "suspend" ? (
              <p>{event.cancellationCompletedAt ? "Cancellation complete" : "Cancellation pending"}</p>
            ) : null}
          </li>
        ))}
      </ul>
      {status && !status.events?.length ? <p>No suspension history.</p> : null}
    </>
  );
}

export function OrganizationSuspensionControl({
  organization,
  onUpdate,
}: {
  organization: { id: string; name: string };
  onUpdate: () => void;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const history = useSuspensionHistory(organization.id);
  const suspended = history.data?.suspended ?? false;
  const cancellationIsPending = cancellationPending(history.data);
  const labels = suspended
    ? {
        status: "Suspended",
        action: "Reactivate organization",
        verb: "Reactivate",
        variant: "outline" as const,
        description:
          "Members can change configuration and start new work after reactivation. Discarded work will not restart.",
      }
    : {
        status: "Active",
        action: "Suspend organization",
        verb: "Suspend",
        variant: "destructive" as const,
        description:
          "Members retain read access. Configuration changes and new work stop. Existing work receives cancellation requests.",
      };
  const validReason = [...reason.trim()].length >= 1 && [...reason.trim()].length <= 2000;
  const changeState = async () => {
    if (pending || (!suspended && !validReason)) return;
    setPending(true);
    setError("");
    try {
      const options = { path: { id: organization.id }, credentials: "include" as const, throwOnError: true as const };
      if (suspended) await adminOrganizationsReactivateOrganization({ ...options, body: {} });
      else await adminOrganizationsSuspendOrganization({ ...options, body: { reason: reason.trim() } });
      await history.refetch();
      void queryClient.invalidateQueries({ queryKey: organizationKeys.all });
      onUpdate();
      setOpen(false);
      setReason("");
    } catch {
      setError(
        "Could not change suspension status. If cancellation is pending, wait for existing work to stop and try again.",
      );
      void history.refetch();
    } finally {
      setPending(false);
    }
  };
  return (
    <section className="space-y-3" aria-label="Organization suspension">
      <h2 className="font-semibold">Status: {labels.status}</h2>
      {history.isError ? (
        <p role="alert">
          Could not load suspension history. <Button onClick={() => void history.refetch()}>Try again</Button>
        </p>
      ) : null}
      {cancellationIsPending ? (
        <p role="status">Cancellation is pending. Reactivation becomes available after existing work stops.</p>
      ) : null}
      <Button
        variant={labels.variant}
        disabled={history.isPending || history.isError || pending || cancellationIsPending}
        onClick={() => {
          setError("");
          setOpen(true);
        }}
      >
        {labels.action}
      </Button>
      <Dialog
        open={open}
        onOpenChange={(value) => {
          if (!pending) setOpen(value);
        }}
      >
        <DialogContent showCloseButton={!pending}>
          <DialogHeader>
            <DialogTitle>
              {labels.verb} “{organization.name}”?
            </DialogTitle>
            <DialogDescription>{labels.description}</DialogDescription>
          </DialogHeader>
          {!suspended ? (
            <div className="space-y-2">
              <Label htmlFor="suspension-reason">Suspension reason</Label>
              <Textarea
                id="suspension-reason"
                value={reason}
                disabled={pending}
                onChange={(event) => setReason(event.target.value)}
                aria-describedby="suspension-reason-help"
              />
              <p id="suspension-reason-help">Enter 1–2000 characters. Only installation admins can read this reason.</p>
            </div>
          ) : null}
          {error ? <p role="alert">{error}</p> : null}
          <DialogFooter>
            <Button variant="outline" disabled={pending} onClick={() => setOpen(false)}>
              Keep current status
            </Button>
            <Button
              disabled={pending || cancellationIsPending || (!suspended && !validReason)}
              onClick={() => void changeState()}
            >
              {labels.action}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <SuspensionHistory status={history.data} />
    </section>
  );
}
