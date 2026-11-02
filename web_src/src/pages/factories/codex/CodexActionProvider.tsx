import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { CodexConnectionSetupDialog } from "./CodexConnectionSetupDialog";
import { CodexActionContext } from "./codexActionContext";

type PendingConnection = {
  organizationId: string;
  factoryId: string;
  stage: "prompt" | "setup";
  resolve: (connected: boolean) => void;
};

export function CodexActionProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<PendingConnection | null>(null);
  const pendingRef = useRef(pending);
  pendingRef.current = pending;
  const requestConnection = useCallback(
    (organizationId: string, factoryId: string) =>
      new Promise<boolean>((resolve) => {
        if (pendingRef.current) {
          resolve(false);
          return;
        }
        const next: PendingConnection = { organizationId, factoryId, stage: "prompt", resolve };
        pendingRef.current = next;
        setPending(next);
      }),
    [],
  );
  const finish = useCallback((connected: boolean) => {
    const current = pendingRef.current;
    pendingRef.current = null;
    setPending(null);
    current?.resolve(connected);
  }, []);
  useEffect(() => () => pendingRef.current?.resolve(false), []);

  return (
    <CodexActionContext.Provider value={requestConnection}>
      {children}
      <Dialog
        open={pending?.stage === "prompt"}
        onOpenChange={(open) => {
          if (!open) finish(false);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <IntegrationIcon integrationName="openai" className="size-9" size={36} />
          <DialogHeader>
            <DialogTitle>Connect Codex to continue</DialogTitle>
            <DialogDescription>
              This workspace uses Codex subscriptions. Connect your Codex account to take ownership of this task.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => finish(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => setPending((current) => (current ? { ...current, stage: "setup" } : null))}
            >
              Connect Codex
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {pending ? (
        <CodexConnectionSetupDialog
          organizationId={pending.organizationId}
          factoryId={pending.factoryId}
          open={pending.stage === "setup"}
          onOpenChange={(open) => {
            if (!open) finish(false);
          }}
          onConnected={() => finish(true)}
        />
      ) : null}
    </CodexActionContext.Provider>
  );
}
