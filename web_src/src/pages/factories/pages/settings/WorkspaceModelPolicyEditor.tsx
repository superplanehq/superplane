import { useId } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";

import { ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";
import { useWorkspaceModelPolicy, type WorkspaceModelPolicyScope } from "./useWorkspaceModelPolicy";

type PolicyEditor = ReturnType<typeof useWorkspaceModelPolicy>;
type PolicyEditorProps = WorkspaceModelPolicyScope & { label: string; canUpdate: boolean; hideEmpty?: boolean };

export function WorkspaceModelPolicyEditor(props: PolicyEditorProps) {
  const editor = useWorkspaceModelPolicy(props);
  const inheritanceId = useId();
  if (editor.query.isLoading)
    return (
      <p role="status" className="text-[13px] text-muted-foreground">
        {COPY.loading}
      </p>
    );
  if (editor.query.isError)
    return <ModelPolicyError label={props.label} onReload={() => void editor.query.refetch()} />;
  if (props.hideEmpty && editor.modelIds.length === 0) return null;
  return (
    <fieldset
      aria-label={`${props.label} models`}
      className="min-w-0 space-y-3"
      data-testid={`llm-models-policy-${props.fundingSource}-${props.provider}`}
    >
      <legend className="mb-3 text-[13px] font-medium">{props.label} models</legend>
      <div className="flex items-center gap-2">
        <Checkbox
          id={inheritanceId}
          checked={editor.policy.inheritParent}
          disabled={!props.canUpdate || editor.saving}
          className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          onChange={(event) => editor.setInheritance(event.currentTarget.checked)}
        />
        <Label htmlFor={inheritanceId} className="text-[13px] font-normal">
          Use all available models
        </Label>
      </div>
      <ModelAllowlistEditor
        modelIds={editor.modelIds}
        modelLabels={editor.labels}
        unavailableModels={editor.configuredIds.filter((id) => !editor.availableIds.includes(id))}
        unavailableLabel={
          props.fundingSource === "subscription" ? "Unavailable in your subscription" : "Unavailable from this provider"
        }
        selected={editor.selected}
        query={editor.search}
        onQueryChange={editor.setSearch}
        onToggle={editor.toggle}
        onBulkToggle={editor.toggleAll}
        disabled={!props.canUpdate || editor.saving || editor.policy.inheritParent}
        searchLabel={`Search ${props.label} models`}
        showCount
        showBulkToggle
      />
      <ModelPolicyAvailability editor={editor} fundingSource={props.fundingSource} />
      <ModelPolicySave editor={editor} canUpdate={props.canUpdate} />
    </fieldset>
  );
}

function ModelPolicyError({ label, onReload }: { label: string; onReload: () => void }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p role="alert" className="text-[13px] text-muted-foreground">
        Unable to load {label} models. Try again.
      </p>
      <Button type="button" size="sm" variant="outline" onClick={onReload}>
        Reload models
      </Button>
    </div>
  );
}

const EMPTY_CATALOG_MESSAGES = {
  subscription: "Connect ChatGPT or refresh your models to load your subscription catalog.",
  hosted: "Ask an organization admin to make models available.",
  byok: "Check the provider integration, then reload the models.",
};

function ModelPolicyAvailability({
  editor,
  fundingSource,
}: {
  editor: PolicyEditor;
  fundingSource: WorkspaceModelPolicyScope["fundingSource"];
}) {
  const selectedUnavailable =
    editor.selected.length > 0 && !editor.selected.some((id) => editor.availableIds.includes(id));
  return (
    <>
      {editor.selected.length === 0 && !editor.policy.inheritParent ? (
        <p role="status" className="text-xs text-muted-foreground">
          No models are enabled. Select models or use all available models.
        </p>
      ) : null}
      {selectedUnavailable ? (
        <p role="status" className="text-xs text-muted-foreground">
          No available models are enabled. Select an available model or use all available models.
        </p>
      ) : null}
      {editor.availableIds.length === 0 ? (
        <p className="text-xs text-muted-foreground">{EMPTY_CATALOG_MESSAGES[fundingSource]}</p>
      ) : null}
    </>
  );
}

function ModelPolicySave({ editor, canUpdate }: { editor: PolicyEditor; canUpdate: boolean }) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <PermissionTooltip allowed={canUpdate} message={COPY.noPermission}>
        <LoadingButton
          type="button"
          size="sm"
          loading={editor.saving}
          loadingText={COPY.saving}
          disabled={!canUpdate || !editor.changed}
          onClick={() => void editor.save()}
        >
          {COPY.save}
        </LoadingButton>
      </PermissionTooltip>
      {editor.saving ? (
        <p role="status" className="text-xs text-muted-foreground">
          Saving models...
        </p>
      ) : null}
      {editor.feedback ? (
        <p role={editor.feedback.error ? "alert" : "status"} className="text-xs text-muted-foreground">
          {editor.feedback.message}
        </p>
      ) : null}
    </div>
  );
}
