import { Link } from "react-router";
import { AlertTriangle } from "lucide-react";
import type { FactoriesFactoryAgentResource, FactoryAutomation } from "@/api-client";
import { Alert, AlertTitle, AlertDescription } from "@/components/reui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";
import { useFactorySkillUsage } from "@/hooks/useFactorySkillUsage";
import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryAppConfigurePath } from "../../lib/factoryPagePaths";
import { skillDisplayTitle } from "./skillFrontmatter";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function SkillDeleteDialog({
  resource,
  canDelete,
  isDeleting,
  onClose,
  onConfirm,
}: {
  resource: FactoriesFactoryAgentResource;
  canDelete: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const usage = useFactorySkillUsage(organizationId, factoryId, resource.name ?? "");
  const inUse = Boolean(usage.data?.length);
  const system = Boolean(resource.systemSkill);
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !isDeleting) onClose();
      }}
    >
      <DialogContent
        role="alertdialog"
        showCloseButton={!isDeleting}
        onInteractOutside={(event) => event.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>
            {system ? "Cannot delete this system skill" : `Delete "${skillDisplayTitle(resource)}"?`}
          </DialogTitle>
          <DialogDescription>
            {system ? "System skills cannot be deleted." : AGENT_RESOURCES_COPY.deleteSkillDescription}
          </DialogDescription>
        </DialogHeader>
        {usage.isFetching ? (
          <p role="status" className="text-sm text-muted-foreground">
            Checking automation usage...
          </p>
        ) : null}
        {usage.isError ? (
          <Alert variant="destructive">
            <AlertDescription>
              Could not check automation usage. Try again before deleting this skill.
              <Button variant="outline" size="sm" onClick={() => void usage.refetch()}>
                Retry
              </Button>
            </AlertDescription>
          </Alert>
        ) : null}
        {inUse ? <SkillUsageWarning resource={resource} automations={usage.data ?? []} /> : null}
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isDeleting}>
            Cancel
          </Button>
          {!system ? (
            <LoadingButton
              variant="destructive"
              loading={isDeleting}
              disabled={!canDelete || !usage.isSuccess || usage.isFetching}
              onClick={() => {
                void onConfirm()
                  .then(onClose)
                  .catch(() => {
                    // The editor reports the error; keep the confirmation open.
                  });
              }}
            >
              {inUse ? "Delete anyway" : "Delete"}
            </LoadingButton>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SkillUsageWarning({
  resource,
  automations,
}: {
  resource: FactoriesFactoryAgentResource;
  automations: FactoryAutomation[];
}) {
  const { organizationId, routeSegment } = useFactoriesLayout();
  return (
    <Alert variant="warning">
      <AlertTriangle />
      <AlertTitle>Used in automations</AlertTitle>
      <AlertDescription>
        <p>
          These automation prompts mention /{resource.name}.
          {!resource.systemSkill ? " Deleting this skill leaves those mentions without a skill to run." : ""}
        </p>
        <ul className="max-h-48 overflow-y-auto list-disc pl-4">
          {automations.map((automation) => (
            <li key={automation.id}>
              <Link
                className="underline underline-offset-2"
                to={factoryAppConfigurePath(organizationId, routeSegment, automation.id ?? "")}
              >
                {automation.name || "Unnamed automation"}
              </Link>
            </li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
