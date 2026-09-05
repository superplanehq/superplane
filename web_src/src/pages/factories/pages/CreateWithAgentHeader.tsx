import { Button } from "@/components/ui/button";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import {
  hostedSelectableLLMModelKey,
  resolveSelectableLLMModelKey,
  selectableLLMModelLabelFromKey,
  type SelectableLLMModel,
} from "@/lib/selectableLLMModels";
import { Loader2, Sparkles } from "lucide-react";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentView } from "./createWithAgentTypes";
import { CreateWithAgentModelPicker } from "./CreateWithAgentModelPicker";

export type CreateWithAgentHeaderProps = {
  workspaceName: string;
  organizationId: string;
  factoryId: string;
  repository: string;
  machineStatus: CreateWithAgentView["machineStatus"];
  selectableModelKey: string;
  onEndSession: () => void;
  onSelectModel?: (key: string) => void;
  modelPickerOpen?: boolean;
  onModelPickerOpenChange?: (open: boolean) => void;
  models?: SelectableLLMModel[];
};

export function CreateWithAgentHeader(props: CreateWithAgentHeaderProps) {
  if (props.models) {
    return <CreateWithAgentHeaderView {...props} models={props.models} isLoading={false} />;
  }
  return <CreateWithAgentHeaderLive {...props} />;
}

function CreateWithAgentHeaderLive(props: Omit<CreateWithAgentHeaderProps, "models">) {
  const query = useSelectableLLMModels(props.organizationId, {
    factoryId: props.factoryId,
    enabled: Boolean(props.organizationId),
  });
  const usage = useOrganizationWorkspaceUsage(props.organizationId);
  const defaultKey = hostedSelectableLLMModelKey(
    usage.data?.defaultHostedProvider ?? "",
    usage.data?.defaultHostedModel ?? "",
  );
  return (
    <CreateWithAgentHeaderView
      {...props}
      models={query.data ?? []}
      isLoading={query.isLoading || usage.isLoading}
      defaultKey={defaultKey}
    />
  );
}

function CreateWithAgentHeaderView({
  workspaceName,
  repository,
  machineStatus,
  selectableModelKey,
  onEndSession,
  onSelectModel,
  modelPickerOpen,
  onModelPickerOpenChange,
  models,
  isLoading,
  defaultKey = "",
}: CreateWithAgentHeaderProps & { models: SelectableLLMModel[]; isLoading: boolean; defaultKey?: string }) {
  const starting = machineStatus === "starting";
  const failed = machineStatus === "failed";
  const selectedKey = resolveSelectableLLMModelKey(selectableModelKey, models, defaultKey);
  const selected = models.find((model) => model.key === selectedKey);
  const selectedLabel =
    selected?.label || selectableLLMModelLabelFromKey(selectedKey) || CREATE_WITH_AGENT_COPY.selectModel;
  return (
    <div
      className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5"
      data-testid="create-with-agent-header"
    >
      <div className="flex min-w-0 items-center gap-2 text-[13px] text-muted-foreground">
        <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted">
          <Sparkles className="size-3" aria-hidden />
        </span>
        <span className="truncate text-foreground">{workspaceName}</span>
        <span aria-hidden>/</span>
        <DialogTitle className="truncate text-[13px] font-medium text-foreground">
          {CREATE_WITH_AGENT_COPY.title}
        </DialogTitle>
        <DialogDescription className="sr-only">Create tasks with an agent in this workspace.</DialogDescription>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <CreateWithAgentModelPicker
          models={models}
          selectedKey={selectedKey}
          selectedLabel={selectedLabel}
          disabled={starting || !onSelectModel}
          loading={isLoading}
          open={modelPickerOpen}
          onOpenChange={onModelPickerOpenChange}
          onSelect={(key) => onSelectModel?.(key)}
        />
        <span
          className="hidden items-center gap-1.5 text-[12px] text-muted-foreground sm:flex"
          data-testid="create-with-agent-machine"
        >
          {starting && !failed ? <Loader2 className="size-3 animate-spin" aria-hidden /> : null}
          {machineStatusLabel(repository, machineStatus)}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 px-2.5 text-[12px]"
          data-testid="create-with-agent-end"
          onClick={onEndSession}
        >
          {CREATE_WITH_AGENT_COPY.endSession}
        </Button>
      </div>
    </div>
  );
}

function machineStatusLabel(repository: string, machineStatus: CreateWithAgentView["machineStatus"]): string {
  if (machineStatus === "starting") {
    return CREATE_WITH_AGENT_COPY.machineStarting;
  }
  if (machineStatus === "failed") {
    return repository
      ? `${repository} · ${CREATE_WITH_AGENT_COPY.machineStopped}`
      : CREATE_WITH_AGENT_COPY.machineStopped;
  }
  const label =
    machineStatus === "waiting" ? CREATE_WITH_AGENT_COPY.machineWaiting : CREATE_WITH_AGENT_COPY.machineRunning;
  return repository ? `${repository} · ${label}` : label;
}
