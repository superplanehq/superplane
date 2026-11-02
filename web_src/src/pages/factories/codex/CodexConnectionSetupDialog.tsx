import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Link } from "@/components/Link/link";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { useFactoryCodexConnection } from "../pages/onboarding/useFactoryCodexConnection";
import { getApiErrorMessage } from "@/lib/errors";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { CodexDeviceSignIn } from "./CodexDeviceSignIn";
import { useCodexConnectionSetup } from "./useCodexConnectionSetup";
import { codexModelLoadError } from "./codexConnectionCopy";

type CodexConnection = ReturnType<typeof useFactoryCodexConnection>;

function CodexConnectionErrorNotice({
  error,
  fallback,
  billingHref,
}: {
  error: unknown;
  fallback: string;
  billingHref: string;
}) {
  const message = getApiErrorMessage(error, fallback);
  const needsHostedCredit = message.startsWith("This organization has no hosted credit.");
  return (
    <p role="alert">
      {message}
      {needsHostedCredit ? (
        <>
          {" "}
          <Link href={billingHref} className="font-medium underline underline-offset-2 hover:no-underline">
            Open Billing
          </Link>
        </>
      ) : null}
    </p>
  );
}

function CodexConnectionNotices({ codex, billingHref }: { codex: CodexConnection; billingHref: string }) {
  const connection = codex.query.data;
  return (
    <>
      {codex.query.isLoading ? <p role="status">Loading Codex connection...</p> : null}
      {codex.query.isError ? (
        <p role="alert">Could not load your Codex connection. Close this dialog and try again.</p>
      ) : null}
      {connection?.revocationPending ? (
        <p role="status">Stopping your previous tasks. You can reconnect when cleanup finishes.</p>
      ) : null}
      {connection?.eligible === false ? <p role="alert">Your account cannot connect Codex in this workspace.</p> : null}
      {codex.start.isError ? (
        <CodexConnectionErrorNotice
          error={codex.start.error}
          fallback="SuperPlane could not start ChatGPT sign-in. Try again."
          billingHref={billingHref}
        />
      ) : null}
      {codex.cancel.isError ? <p role="alert">Could not cancel Codex sign-in. Try again.</p> : null}
      {codex.refreshCatalog.isError ? (
        <p role="alert">{codexModelLoadError(codex.refreshCatalog.error, connection?.status)}</p>
      ) : null}
    </>
  );
}

export function CodexConnectionSetupContent({ setup }: { setup: ReturnType<typeof useCodexConnectionSetup> }) {
  const { codex } = setup;
  const billingHref = factorySettingsSectionPath(setup.organizationId, setup.factoryId, "organization", "billing");
  return (
    <>
      <CodexConnectionNotices codex={codex} billingHref={billingHref} />
      <CodexDeviceSignIn
        disabled={setup.disabled}
        connection={codex.query.data}
        starting={codex.start.isPending}
        canceling={codex.cancel.isPending}
        onStart={() => codex.start.mutate()}
        onCancel={() => void setup.closeSetup()}
        cancelAction={null}
      />
      {setup.completionError ? (
        <div>
          <p role="alert">{setup.completionError}</p>
          <Button type="button" disabled={setup.busy} onClick={setup.retryCompletion}>
            Try again
          </Button>
        </div>
      ) : null}
      <DialogFooter>
        <LoadingButton
          type="button"
          variant="outline"
          disabled={setup.busy}
          loading={codex.cancel.isPending}
          onClick={() => void setup.closeSetup()}
        >
          Cancel
        </LoadingButton>
      </DialogFooter>
    </>
  );
}

export function CodexConnectionSetupDialog({
  organizationId,
  factoryId,
  open,
  onOpenChange,
  onConnected,
}: {
  organizationId: string;
  factoryId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConnected?: () => void | Promise<void>;
}) {
  const setup = useCodexConnectionSetup({
    organizationId,
    factoryId,
    open,
    onConnected,
    onClose: () => onOpenChange(false),
  });
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen && !setup.busy) void setup.closeSetup();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Connect Codex</DialogTitle>
          <DialogDescription>Connect once to use Codex across this organization’s workspaces.</DialogDescription>
        </DialogHeader>
        {open ? <CodexConnectionSetupContent setup={setup} /> : null}
      </DialogContent>
    </Dialog>
  );
}
