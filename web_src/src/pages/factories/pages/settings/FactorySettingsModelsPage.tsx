import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { BYOK_PROVIDERS, useFactoryLLMModels, useUpdateFactoryLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { filterModelIds } from "@/lib/hostedLLMModels";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";
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
  const selected = draft ?? (factoryModels.data?.selected ?? []).map((model) => model.id ?? "").filter(Boolean);
  const visibleModels = useMemo(() => filterModelIds(parent, query), [parent, query]);

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

  return (
    <div>
      <p className="text-[13px] font-medium">{title}</p>
      {factoryModels.isLoading ? (
        <Text className="mt-2 text-[13px] text-muted-foreground">Loading models...</Text>
      ) : parent.length === 0 ? (
        <Text className="mt-2 text-[13px] text-muted-foreground">
          {fundingSource === "hosted"
            ? "No SuperPlane-hosted models are available."
            : "No organization models are selected for this provider."}
        </Text>
      ) : (
        <div className="mt-3 space-y-3">
          <div className="flex items-center justify-between gap-3">
            <Label className="text-[13px]">Use organization list</Label>
            <Switch
              checked={inheritParent}
              disabled={!canUpdate || update.isPending}
              onCheckedChange={(checked) => {
                setInherit(checked);
                if (!checked) {
                  setDraft(selected.length > 0 ? selected : parent);
                }
              }}
            />
          </div>
          {inheritParent ? (
            <Text className="text-[13px] text-muted-foreground">This workspace uses the organization model list.</Text>
          ) : (
            <>
              <InputGroup className="relative">
                <Search
                  size={14}
                  className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                />
                <Input
                  type="search"
                  className="pl-9"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Search models..."
                  aria-label={`Search ${title}`}
                />
              </InputGroup>
              <div className="max-h-56 space-y-2 overflow-auto">
                {visibleModels.map((model) => (
                  <label key={model} className="flex items-center gap-2 text-[13px]">
                    <Checkbox
                      checked={selected.includes(model)}
                      disabled={!canUpdate || update.isPending}
                      onChange={(event) => {
                        const checked = event.currentTarget.checked;
                        setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model));
                      }}
                    />
                    <span className="font-mono text-xs">{model}</span>
                  </label>
                ))}
              </div>
            </>
          )}
          {canUpdate ? (
            <PermissionTooltip allowed={canUpdate} message="You do not have permission to update workspace models.">
              <Button type="button" onClick={() => void save()} disabled={update.isPending}>
                {update.isPending ? "Saving..." : "Save models"}
              </Button>
            </PermissionTooltip>
          ) : null}
        </div>
      )}
    </div>
  );
}
