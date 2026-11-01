import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useRunnerRegistration } from "@/hooks/useRunnerRegistration";
import type { OrganizationsCreateFleetRunnerResponse } from "@/api-client";

const shellQuote = (value: string) => `'${value.replace(/'/g, "'\\''")}'`;

export function ConnectRunner({
  organizationId,
  fleetId,
  onClose,
  refresh,
}: {
  organizationId: string;
  fleetId: string;
  onClose: () => void;
  refresh: () => void;
}) {
  const { mode, registration, status, error, setError, create, chooseMode, retryStatus } = useRunnerRegistration({
    organizationId,
    fleetId,
    refresh,
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Connect runner</DialogTitle>
          <DialogDescription>
            Run a compatible runner executable on your machine. Registration does not start a process or provision
            infrastructure.
          </DialogDescription>
        </DialogHeader>
        <p className="text-sm">
          An ephemeral runner exits after one task. A reusable runner waits for successive tasks.
        </p>
        {status === "ready" && (
          <>
            <Label htmlFor="runner-mode">Lifecycle mode</Label>
            <Select value={mode} disabled={!!error} onValueChange={chooseMode}>
              <SelectTrigger id="runner-mode">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ephemeral">Ephemeral</SelectItem>
                <SelectItem value="reusable">Reusable</SelectItem>
              </SelectContent>
            </Select>
            <Button onClick={() => void create()}>{error ? "Retry registration" : "Generate registration"}</Button>
          </>
        )}
        {status === "creating" && <p role="status">Creating registration…</p>}
        {registration && <RegistrationOutput registration={registration} setError={setError} />}
        {status === "connected" && <p role="status">Runner connected. Registration confirmed by the server.</p>}
        {status === "expired" && (
          <p role="status">Registration expired. Create a new registration and use its command.</p>
        )}
        {status === "terminated" && (
          <p role="status">Runner terminated before registration. Create a new registration.</p>
        )}
        {(status === "expired" || status === "terminated") && (
          <Button
            onClick={() => {
              void create(true);
            }}
          >
            Create new registration
          </Button>
        )}
        {error && <p role="alert">{error}</p>}
        {error && status === "waiting" && (
          <Button variant="outline" onClick={retryStatus}>
            Retry status check
          </Button>
        )}
        <Button variant="outline" onClick={onClose}>
          Close setup
        </Button>
      </DialogContent>
    </Dialog>
  );
}

function RegistrationOutput({
  registration,
  setError,
}: {
  registration: OrganizationsCreateFleetRunnerResponse;
  setError: (message: string) => void;
}) {
  const [copied, setCopied] = useState(false);
  const expiresAt = Date.parse(registration.registrationExpiresAt!);
  const [expired, setExpired] = useState(Date.now() >= expiresAt);
  useEffect(() => {
    if (expired) return;
    const timer = setTimeout(() => setExpired(true), Math.max(0, expiresAt - Date.now()));
    return () => clearTimeout(timer);
  }, [expiresAt, expired]);
  const command = `runner --url ${shellQuote(registration.runnerApiUrl!)} --registration-token ${shellQuote(registration.registrationToken!)}`;
  const copy = async () => {
    if (Date.now() >= expiresAt) return;
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
    } catch {
      setError("Could not copy the command. Copy it manually.");
    }
  };
  return (
    <>
      <p role="status">Waiting for runner registration.</p>
      <p>Expires: {new Date(expiresAt).toLocaleString()}</p>
      {!expired ? (
        <>
          <pre className="whitespace-pre-wrap break-all rounded border p-3 text-xs">{command}</pre>
          <Button onClick={() => void copy()}>{copied ? "Copied" : "Copy command"}</Button>
        </>
      ) : (
        <p>Checking final registration state before expiration.</p>
      )}
    </>
  );
}
