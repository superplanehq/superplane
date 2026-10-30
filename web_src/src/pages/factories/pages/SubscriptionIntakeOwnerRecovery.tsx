import { useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { canvasesDescribeCanvas, type CanvasesCanvas, type FactoriesFactoryIntake } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { usePermissions } from "@/contexts/usePermissions";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useFactoryAutomations } from "@/hooks/useFactoryData";
import { useFactoryIntakes, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { CodexConnectionSetupDialog } from "../codex/CodexConnectionSetupDialog";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";
import { ScheduledIntakeOwnerSettings } from "./ScheduledIntakeOwnerSettings";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";

export function SubscriptionIntakeOwnerRecovery() {
  const layout = useOptionalFactoriesLayout();
  if (!layout?.factory?.onboarding?.codexSubscription) return null;
  return <IntakeOwnerRecovery organizationId={layout.organizationId} factoryId={layout.factoryId} />;
}

type RepairTarget = { id: string; name: string; intake?: FactoriesFactoryIntake; canvas?: CanvasesCanvas };

function IntakeOwnerRecovery({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  const { currentUserId, canAct } = usePermissions();
  const recovery = useIntakeOwnerRecoveryData(organizationId, factoryId);
  const [setupOpen, setSetupOpen] = useState(false);
  const [connectOpen, setConnectOpen] = useState(false);
  const [connectedOwnerId, setConnectedOwnerId] = useState<string>();
  const [selectedId, setSelectedId] = useState<string>();
  const { affected, loading, failed, connectedMembers } = recovery;
  const target = affected.find((item) => item.id === selectedId) ?? affected[0];
  if (loading || (!failed && affected.length === 0)) return null;
  return (
    <>
      <IntakeOwnerRecoveryNotice
        failed={failed}
        connectedMembers={connectedMembers}
        canUpdate={canAct("factories", "update")}
        onRetry={recovery.retry}
        onSetup={() => (connectedMembers ? setSetupOpen(true) : setConnectOpen(true))}
      />
      <CodexConnectionSetupDialog
        organizationId={organizationId}
        factoryId={factoryId}
        open={connectOpen}
        onOpenChange={setConnectOpen}
        onConnected={async () => {
          await recovery.refetchEligibility();
          setConnectedOwnerId(currentUserId);
          setConnectOpen(false);
          setSetupOpen(true);
        }}
      />
      <Dialog open={setupOpen} onOpenChange={setSetupOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Set default owner</DialogTitle>
            <DialogDescription>Select an owner with Codex connected for each intake.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-wrap gap-2">
            {affected.map((item) => (
              <Button
                key={item.id}
                size="sm"
                variant={target?.id === item.id ? "secondary" : "outline"}
                onClick={() => setSelectedId(item.id)}
              >
                {item.name}
              </Button>
            ))}
          </div>
          {target?.intake ? (
            <ExternalIntakeOwnerRepair
              key={target.id}
              organizationId={organizationId}
              factoryId={factoryId}
              intake={target.intake}
              initialOwnerId={connectedOwnerId}
            />
          ) : null}
          {target?.canvas ? (
            <ScheduledIntakeOwnerRepair
              key={target.id}
              organizationId={organizationId}
              factoryId={factoryId}
              canvas={target.canvas}
              initialOwnerId={connectedOwnerId}
            />
          ) : null}
          {!target ? <p>All intake owners are set.</p> : null}
        </DialogContent>
      </Dialog>
    </>
  );
}

function ExternalIntakeOwnerRepair({
  organizationId,
  factoryId,
  intake,
  initialOwnerId,
}: {
  organizationId: string;
  factoryId: string;
  intake: FactoriesFactoryIntake;
  initialOwnerId?: string;
}) {
  const [ownerId, setOwnerId] = useState(initialOwnerId ?? "");
  const update = useUpdateFactoryIntake(organizationId, factoryId);
  return (
    <div className="space-y-4">
      <IntakeDefaultOwnerField
        organizationId={organizationId}
        factoryId={factoryId}
        required
        value={ownerId}
        onChange={setOwnerId}
        disabled={update.isPending}
      />
      {update.error ? (
        <p role="alert" className="text-sm text-destructive">
          {getApiErrorMessage(update.error, "SuperPlane could not save the default owner. Try again.")}
        </p>
      ) : null}
      <Button
        disabled={!ownerId || update.isPending}
        onClick={() =>
          void update
            .mutateAsync({ intakeId: intake.id!, settings: { ...intake.settings, defaultOwnerId: ownerId } })
            .catch(() => {})
        }
      >
        {update.isPending ? "Saving…" : "Save default owner"}
      </Button>
    </div>
  );
}

function ScheduledIntakeOwnerRepair({
  organizationId,
  factoryId,
  canvas,
  initialOwnerId,
}: {
  organizationId: string;
  factoryId: string;
  canvas: CanvasesCanvas;
  initialOwnerId?: string;
}) {
  const editor = useColumnCanvasAgentEditor(organizationId, canvas.metadata?.id);
  const node = canvas.spec?.nodes?.find((node) => node.component === "createWorkOrder");
  if (!node?.id) return <p role="alert">Open the intake canvas and restore its task creation step.</p>;
  return (
    <ScheduledIntakeOwnerSettings
      organizationId={organizationId}
      factoryId={factoryId}
      node={node}
      initialOwnerId={initialOwnerId}
      onSave={(update) => editor.saveNode(update, { notify: false })}
    />
  );
}

function useIntakeOwnerRecoveryData(organizationId: string, factoryId: string) {
  const eligibility = useFactoryOwnerEligibility(organizationId, factoryId);
  const intakes = useFactoryIntakes(organizationId, factoryId);
  const apps = useFactoryAutomations(organizationId, factoryId);
  const schedules = (apps.data ?? []).filter((app) => app.id && app.templateId === "scheduled-intake");
  const canvases = useQueries({
    queries: schedules.map((app) => ({
      queryKey: canvasKeys.detail(organizationId, app.id!),
      queryFn: async () => {
        const response = await canvasesDescribeCanvas(
          withOrganizationHeader({ organizationId, path: { id: app.id! } }),
        );
        if (!response.data?.canvas) throw new Error("Could not load intake settings.");
        return response.data.canvas;
      },
    })),
  });
  const members = eligibility.data?.members ?? [];
  const validOwner = (id: string | undefined) => Boolean(id && members.some((member) => member.userId === id));
  const affected: RepairTarget[] = (intakes.data ?? [])
    .filter((intake) => !validOwner(intake.settings?.defaultOwnerId))
    .map((intake) => ({ id: intake.id!, name: intake.name ?? "Intake", intake }));
  canvases.forEach((query, index) => {
    if (!query.data) return;
    if (validOwner(scheduledIntakeOwnerId(query.data))) return;
    affected.push({ id: schedules[index].id!, name: schedules[index].name ?? "Scheduled intake", canvas: query.data });
  });
  const connectedMembers = members.some((member) => member.eligible);
  const queries = [eligibility, intakes, apps, ...canvases];
  const loading = queries.some((query) => query.isPending);
  const failed = queries.some((query) => query.isError);

  const retry = () => {
    void eligibility.refetch();
    void intakes.refetch();
    void apps.refetch();
    canvases.forEach((query) => void query.refetch());
  };
  return { affected, loading, failed, connectedMembers, retry, refetchEligibility: eligibility.refetch };
}

function IntakeOwnerRecoveryNotice({
  failed,
  connectedMembers,
  canUpdate,
  onRetry,
  onSetup,
}: {
  failed: boolean;
  connectedMembers: boolean;
  canUpdate: boolean;
  onRetry: () => void;
  onSetup: () => void;
}) {
  return (
    <div
      role="status"
      className="flex flex-wrap items-center gap-3 border-b border-border bg-muted/50 px-4 py-3 text-sm"
    >
      <p className="min-w-0 flex-1">
        {failed
          ? "SuperPlane could not check intake owners."
          : "Set a default owner for intake tasks before they can run."}
      </p>
      {failed ? (
        <Button size="sm" variant="outline" onClick={onRetry}>
          Try again
        </Button>
      ) : canUpdate ? (
        <Button size="sm" variant="outline" onClick={onSetup}>
          {connectedMembers ? "Set default owner" : "Connect Codex"}
        </Button>
      ) : (
        <p>Ask a workspace admin to set the default owner.</p>
      )}
    </div>
  );
}

function scheduledIntakeOwnerId(canvas: CanvasesCanvas): string | undefined {
  const creators = canvas.spec?.nodes?.filter((node) => node.component === "createWorkOrder");
  if (creators?.length !== 1) return undefined;
  const ownerId = creators[0].configuration?.defaultOwnerId;
  return typeof ownerId === "string" ? ownerId : undefined;
}
