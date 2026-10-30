import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { useFactoryCodexConnection } from "../pages/onboarding/useFactoryCodexConnection";
import { CodexDeviceSignIn } from "./CodexDeviceSignIn";
import { useCodexConnectionSetup } from "./useCodexConnectionSetup";

type CodexConnection = ReturnType<typeof useFactoryCodexConnection>;

function CodexConnectionNotices({
  codex,
  featuresLoading,
  offered,
}: {
  codex: CodexConnection;
  featuresLoading: boolean;
  offered: boolean;
}) {
  const connection = codex.query.data;
  return (
    <>
      {featuresLoading || codex.query.isLoading ? <p role="status">Loading Codex connection...</p> : null}
      {codex.query.isError ? (
        <p role="alert">Could not load your Codex connection. Close this dialog and try again.</p>
      ) : null}
      {connection?.revocationPending ? (
        <p role="status">Stopping your previous tasks. You can reconnect when cleanup finishes.</p>
      ) : null}
      {!featuresLoading && !offered ? <p role="alert">Codex subscriptions are unavailable in this workspace.</p> : null}
      {connection?.eligible === false ? <p role="alert">Your account cannot connect Codex in this workspace.</p> : null}
      {codex.start.isError ? <p role="alert">Could not start Codex sign-in. Try again.</p> : null}
      {codex.cancel.isError ? <p role="alert">Could not cancel Codex sign-in. Try again.</p> : null}
      {codex.refreshCatalog.isError ? (
        <p role="alert">Could not load your Codex models. Close this dialog and try again.</p>
      ) : null}
    </>
  );
}

export function CodexConnectionSetupContent({ setup }: { setup: ReturnType<typeof useCodexConnectionSetup> }) {
  const { codex } = setup;
  return (
    <>
      <CodexConnectionNotices codex={codex} featuresLoading={setup.featuresLoading} offered={setup.offered} />
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
