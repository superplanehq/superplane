import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { BYOK_PROVIDERS, useFactoryLLMModels, useUpdateFactoryLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";
import { useState } from "react";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

const PROVIDERS = [...BYOK_PROVIDERS];

export function FactorySettingsModelsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update");

  usePageTitle(["Models", "Settings", factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      title="Models"
      subtitle="Limit SuperPlane-hosted and your-key models for this workspace. An empty list uses the organization list."
    >
      {PROVIDERS.map((provider) => (
        <FactorySettingsCard key={provider} title={hostedProviderLabel(provider)}>
          <FactoryProviderModelSection
            organizationId={organizationId}
            factoryId={factoryId}
            provider={provider}
            fundingSource="hosted"
            title="SuperPlane-hosted models"
            canUpdate={canUpdate && !permissionsLoading}
          />
          <div className="mt-6 border-t border-border pt-4">
            <FactoryProviderModelSection
              organizationId={organizationId}
              factoryId={factoryId}
              provider={provider}
              fundingSource="byok"
              title="Use your keys"
              canUpdate={canUpdate && !permissionsLoading}
            />
          </div>
        </FactorySettingsCard>
      ))}
    </FactorySettingsPageFrame>
  );
}

function FactoryProviderModelSection({
  organizationId,
  factoryId,
  provider,
  fundingSource,
  title,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  provider: string;
  fundingSource: "hosted" | "byok";
  title: string;
  canUpdate: boolean;
}) {
  const factoryModels = useFactoryLLMModels(organizationId, factoryId, provider, fundingSource, true);
  const update = useUpdateFactoryLLMModels(organizationId, factoryId);
  const [query, setQuery] = useState("");
  const [inherit, setInherit] = useState<boolean | null>(null);
  const [draft, setDraft] = useState<string[] | null>(null);

  const parent = (factoryModels.data?.parent ?? []).map((model) => model.id ?? "").filter(Boolean);
  const inheritParent = inherit ?? factoryModels.data?.inheritParent !== false;
  const selected =
    draft ??
    (factoryModels.data?.selected ?? [])
      .map((model) => model.id ?? "")
      .filter((id) => id !== "" && parent.includes(id));

  const save = async () => {
    try {
      await update.mutateAsync({
        provider,
        fundingSource,
        allowedModels: inheritParent ? [] : selected,
      });
      setDraft(null);
      setInherit(null);
      showSuccessToast("Workspace models saved.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Unable to save workspace models."));
    }
  };

  const emptyMessage =
    fundingSource === "hosted"
      ? "No SuperPlane-hosted models are available."
      : "No organization models are selected for this provider.";

  return (
    <div>
      <p className="text-[13px] font-medium">{title}</p>
      <FactoryModelSectionBody
        canUpdate={canUpdate}
        emptyMessage={emptyMessage}
        inheritParent={inheritParent}
        isLoading={factoryModels.isLoading}
        isPending={update.isPending}
        parent={parent}
        query={query}
        selected={selected}
        title={title}
        onInheritChange={(checked) => {
          setInherit(checked);
          if (!checked) {
            setDraft(selected.length > 0 ? selected : parent);
          }
        }}
        onQueryChange={setQuery}
        onSave={() => void save()}
        onToggle={(model, checked) => setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model))}
      />
    </div>
  );
}

function FactoryModelSectionBody({
  canUpdate,
  emptyMessage,
  inheritParent,
  isLoading,
  isPending,
  parent,
  query,
  selected,
  title,
  onInheritChange,
  onQueryChange,
  onSave,
  onToggle,
}: {
  canUpdate: boolean;
  emptyMessage: string;
  inheritParent: boolean;
  isLoading: boolean;
  isPending: boolean;
  parent: string[];
  query: string;
  selected: string[];
  title: string;
  onInheritChange: (checked: boolean) => void;
  onQueryChange: (query: string) => void;
  onSave: () => void;
  onToggle: (model: string, checked: boolean) => void;
}) {
  if (isLoading) {
    return <Text className="mt-2 text-[13px] text-muted-foreground">Loading models...</Text>;
  }
  if (parent.length === 0) {
    return <Text className="mt-2 text-[13px] text-muted-foreground">{emptyMessage}</Text>;
  }

  return (
    <div className="mt-3 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <Label className="text-[13px]">Use organization list</Label>
        <Switch checked={inheritParent} disabled={!canUpdate || isPending} onCheckedChange={onInheritChange} />
      </div>
      {inheritParent ? (
        <Text className="text-[13px] text-muted-foreground">This workspace uses the organization model list.</Text>
      ) : (
        <ModelAllowlistEditor
          modelIds={parent}
          selected={selected}
          query={query}
          onQueryChange={onQueryChange}
          onToggle={onToggle}
          disabled={!canUpdate || isPending}
          searchLabel={`Search ${title}`}
        />
      )}
      {canUpdate ? (
        <PermissionTooltip allowed={canUpdate} message="You do not have permission to update workspace models.">
          <Button type="button" onClick={onSave} disabled={isPending}>
            {isPending ? "Saving..." : "Save models"}
          </Button>
        </PermissionTooltip>
      ) : null}
    </div>
  );
}
