import { useState } from "react";
import { Check, Plug, TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import type { OrganizationsIntegration } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { IntegrationChoiceIcon } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { useFactoryRepository } from "./useFactoryRepository";

type ConnectionWorkspace = {
  onboarding?: {
    appRepository?: string;
    defaultBranch?: string;
    vcsProvider?: string;
    vcsIntegrationId?: string;
  } | null;
};

type CardProps = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: ConnectionWorkspace | null | undefined;
  canUpdate: boolean;
};

export function WorkspaceGitConnectionCard(props: CardProps) {
  const { canAct } = usePermissions();
  const repository = props.factory?.onboarding?.appRepository?.trim() ?? "";
  if (!repository || !canAct("integrations", "read")) return null;
  return <GitConnectionCard {...props} repository={repository} />;
}

function integrationName(integration: OrganizationsIntegration | undefined): string {
  return integration?.metadata?.name || integration?.metadata?.id || "";
}

function GitConnectionCard(props: CardProps & { repository: string }) {
  const [choosing, setChoosing] = useState(false);
  const onboarding = props.factory?.onboarding;
  const host: VcsHostId = onboarding?.vcsProvider === "bitbucket" ? "bitbucket" : "github";
  const currentId = onboarding?.vcsIntegrationId ?? "";
  const integrations = useConnectedIntegrations(props.organizationId);
  const hostIntegrations = (integrations.data ?? []).filter(
    (integration) => integration.metadata?.integrationName === host && Boolean(integration.metadata?.id),
  );
  const current = hostIntegrations.find((integration) => integration.metadata?.id === currentId);
  const choices = hostIntegrations.filter((integration) => integration.status?.state === "ready");
  const manageHref = factorySettingsSectionPath(props.organizationId, props.factoryKey, "organization", "integrations");

  return (
    <FactorySettingsCard
      title={AGENT_RESOURCES_COPY.gitConnectionTitle}
      titleClassName="text-sm font-semibold"
      description={AGENT_RESOURCES_COPY.gitConnectionHelper}
      icon={<Plug className="size-4" strokeWidth={1.75} aria-hidden />}
      iconClassName="border-border bg-muted/60 text-foreground"
      className="shadow-sm"
      attachedList
      action={
        <PermissionTooltip allowed={props.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button
            type="button"
            size="sm"
            disabled={!props.canUpdate}
            onClick={() => setChoosing(true)}
            data-testid="agent-git-connection-change"
          >
            {current ? AGENT_RESOURCES_COPY.gitConnectionChange : AGENT_RESOURCES_COPY.gitConnectionChoose}
          </Button>
        </PermissionTooltip>
      }
      data-testid="agent-git-connection"
    >
      <div className="border-t border-border py-3">
        {current ? (
          <CurrentConnectionRow host={host} integration={current} />
        ) : (
          <p className="flex items-start gap-2 text-[13px] text-amber-700 dark:text-amber-400">
            <TriangleAlert className="mt-0.5 size-4 shrink-0" strokeWidth={1.75} aria-hidden />
            {AGENT_RESOURCES_COPY.gitConnectionMissing}
          </p>
        )}
      </div>
      <ChooseConnectionDialog
        key={`${choosing}:${currentId}`}
        {...props}
        open={choosing}
        onOpenChange={setChoosing}
        host={host}
        currentId={currentId}
        choices={choices}
        manageHref={manageHref}
      />
    </FactorySettingsCard>
  );
}

function CurrentConnectionRow({ host, integration }: { host: VcsHostId; integration: OrganizationsIntegration }) {
  const ready = integration.status?.state === "ready";
  return (
    <div className="flex min-w-0 items-center gap-2.5" data-testid="agent-git-connection-current">
      <IntegrationChoiceIcon name={host} />
      <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{integrationName(integration)}</span>
      <span
        className={cn(
          "shrink-0 text-[12px]",
          ready ? "text-emerald-600 dark:text-emerald-500" : "text-amber-600 dark:text-amber-500",
        )}
      >
        {ready ? AGENT_RESOURCES_COPY.gitConnectionReady : AGENT_RESOURCES_COPY.gitConnectionNeedsAttention}
      </span>
    </div>
  );
}

function ChooseConnectionDialog(
  props: CardProps & {
    repository: string;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    host: VcsHostId;
    currentId: string;
    choices: OrganizationsIntegration[];
    manageHref: string;
  },
) {
  const [selectedId, setSelectedId] = useState(props.currentId);
  const updateRepository = useFactoryRepository(props.organizationId, props.factoryId);

  const save = async () => {
    try {
      await updateRepository.mutateAsync({
        repository: props.repository,
        defaultBranch: props.factory?.onboarding?.defaultBranch?.trim() ?? "",
        integrationId: selectedId,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.gitConnectionUpdated);
      props.onOpenChange(false);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.gitConnectionUpdateFailed));
    }
  };

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="agent-git-connection-dialog">
        <DialogHeader>
          <DialogTitle>{AGENT_RESOURCES_COPY.gitConnectionDialogTitle}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.gitConnectionDialogHelper}</DialogDescription>
        </DialogHeader>
        {props.choices.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border px-3 py-6 text-center text-[13px] text-muted-foreground">
            {AGENT_RESOURCES_COPY.gitConnectionEmpty}
          </p>
        ) : (
          <ConnectionOptions {...props} selectedId={selectedId} onSelect={setSelectedId} />
        )}
        <div className="flex items-center justify-between gap-3 pt-2">
          <Link to={props.manageHref} className="text-[13px] text-muted-foreground underline underline-offset-2">
            {AGENT_RESOURCES_COPY.gitConnectionManage}
          </Link>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={() => props.onOpenChange(false)}>
              {AGENT_RESOURCES_COPY.cancel}
            </Button>
            <LoadingButton
              type="button"
              loading={updateRepository.isPending}
              disabled={!selectedId || selectedId === props.currentId}
              onClick={() => void save()}
              data-testid="agent-git-connection-save"
            >
              {AGENT_RESOURCES_COPY.gitConnectionSave}
            </LoadingButton>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ConnectionOptions(props: {
  host: VcsHostId;
  currentId: string;
  choices: OrganizationsIntegration[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <div role="radiogroup" aria-label={AGENT_RESOURCES_COPY.gitConnectionTitle} className="flex flex-col gap-2">
      {props.choices.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === props.selectedId;
        return (
          <button
            key={id}
            type="button"
            role="radio"
            aria-checked={selected}
            onClick={() => props.onSelect(id)}
            className={cn(
              "flex min-w-0 items-center gap-2.5 rounded-lg border px-3 py-2.5 text-left transition-colors",
              selected ? "border-foreground/40 bg-accent" : "border-border hover:bg-accent/40",
            )}
            data-testid={`agent-git-connection-option-${id}`}
          >
            <IntegrationChoiceIcon name={props.host} />
            <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{integrationName(integration)}</span>
            {id === props.currentId ? (
              <span className="shrink-0 text-[12px] text-muted-foreground">
                {AGENT_RESOURCES_COPY.gitConnectionCurrent}
              </span>
            ) : null}
            <Check className={cn("size-4 shrink-0", selected ? "opacity-100" : "opacity-0")} aria-hidden />
          </button>
        );
      })}
    </div>
  );
}
