import { TriangleAlert } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { BYOK_PROVIDERS, useBYOKLLMModels, useUpdateBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  byokProviderProductName,
  disconnectedProviderMessage,
  ORGANIZATION_LLM_MODELS_COPY as COPY,
  shouldShowLLMModelsLandingBanner,
} from "./organizationLLMModelsCopy";

const PROVIDERS = [...BYOK_PROVIDERS];

export function FactorySettingsOrganizationLLMModelsPage() {
  const { organizationId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("org", "update") && !permissionsLoading;
  const integrationsHref = useIntegrationsBasePath(organizationId);
  const anthropic = useBYOKLLMModels(organizationId, "anthropic", true);
  const openai = useBYOKLLMModels(organizationId, "openai", true);
  const openrouter = useBYOKLLMModels(organizationId, "openrouter", true);
  const showLanding = shouldShowLLMModelsLandingBanner([anthropic, openai, openrouter]);

  usePageTitle([COPY.pageTitle, "Settings", factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame title={COPY.pageTitle} subtitle={COPY.pageSubtitle}>
      <div data-testid="factory-settings-llm-models" className="flex flex-col gap-5">
        {showLanding ? <LLMModelsLandingBanner integrationsHref={integrationsHref} /> : null}
        {PROVIDERS.map((provider) => (
          <FactorySettingsCard
            key={provider}
            title={byokProviderProductName(provider)}
            data-testid={`factory-settings-llm-models-${provider}`}
          >
            <BYOKProviderAllowlist
              organizationId={organizationId}
              provider={provider}
              canUpdate={canUpdate}
              integrationsHref={integrationsHref}
            />
          </FactorySettingsCard>
        ))}
      </div>
    </FactorySettingsPageFrame>
  );
}

function LLMModelsLandingBanner({ integrationsHref }: { integrationsHref: string }) {
  return (
    <div
      role="status"
      data-testid="llm-models-empty-banner"
      className="flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50/90 px-3.5 py-2.5 text-sm text-amber-950 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100"
    >
      <div className="flex min-w-0 items-start gap-2">
        <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
        <p>
          <span className="font-medium">{COPY.landingTitle}. </span>
          {COPY.landingDescription}
        </p>
      </div>
      <Button
        asChild
        variant="outline"
        size="sm"
        className="border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60"
      >
        <Link to={integrationsHref}>{COPY.landingAction}</Link>
      </Button>
    </div>
  );
}

function BYOKProviderAllowlist({
  organizationId,
  provider,
  canUpdate,
  integrationsHref,
}: {
  organizationId: string;
  provider: string;
  canUpdate: boolean;
  integrationsHref: string;
}) {
  const { data, isLoading, error } = useBYOKLLMModels(organizationId, provider, true);
  const update = useUpdateBYOKLLMModels(organizationId);
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<string[] | null>(null);
  const selected = draft ?? (data?.selected ?? []).map((model) => model.id ?? "").filter(Boolean);
  const candidates = (data?.candidates ?? []).map((model) => model.id ?? "").filter(Boolean);
  const modelIds = candidates.length > 0 ? candidates : selected;

  const save = async () => {
    try {
      await update.mutateAsync({ provider, allowedModels: selected });
      setDraft(null);
      showSuccessToast(COPY.saveSuccess);
    } catch (saveError) {
      showErrorToast(getApiErrorMessage(saveError, COPY.saveError));
    }
  };

  return (
    <BYOKAllowlistBody
      canUpdate={canUpdate}
      connected={Boolean(data?.connected)}
      error={Boolean(error)}
      integrationsHref={integrationsHref}
      isLoading={isLoading}
      isPending={update.isPending}
      modelIds={modelIds}
      provider={provider}
      query={query}
      selected={selected}
      showSave={draft !== null}
      onQueryChange={setQuery}
      onSave={() => void save()}
      onToggle={(model, checked) => setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model))}
    />
  );
}

function BYOKAllowlistBody({
  canUpdate,
  connected,
  error,
  integrationsHref,
  isLoading,
  isPending,
  modelIds,
  provider,
  query,
  selected,
  showSave,
  onQueryChange,
  onSave,
  onToggle,
}: {
  canUpdate: boolean;
  connected: boolean;
  error: boolean;
  integrationsHref: string;
  isLoading: boolean;
  isPending: boolean;
  modelIds: string[];
  provider: string;
  query: string;
  selected: string[];
  showSave: boolean;
  onQueryChange: (query: string) => void;
  onSave: () => void;
  onToggle: (model: string, checked: boolean) => void;
}) {
  if (isLoading) {
    return <Text className="text-[13px] text-muted-foreground">{COPY.loading}</Text>;
  }
  if (error) {
    return <Text className="text-[13px] text-destructive">{COPY.listError}</Text>;
  }
  if (!connected) {
    return (
      <div className="space-y-2">
        <Text className="text-[13px] text-muted-foreground">{disconnectedProviderMessage(provider)}</Text>
        <Link to={integrationsHref} className="inline-block text-[13px] text-foreground underline underline-offset-2">
          {COPY.disconnectedLink}
        </Link>
      </div>
    );
  }
  if (modelIds.length === 0) {
    return <Text className="text-[13px] text-muted-foreground">{COPY.emptyCatalog}</Text>;
  }

  return (
    <div className="space-y-3">
      <ModelAllowlistEditor
        modelIds={modelIds}
        selected={selected}
        query={query}
        onQueryChange={onQueryChange}
        onToggle={onToggle}
        disabled={!canUpdate || isPending}
        searchLabel={`Search ${byokProviderProductName(provider)} models`}
        showCount
      />
      <PermissionTooltip allowed={canUpdate} message={COPY.noPermission}>
        <Button type="button" onClick={onSave} disabled={!canUpdate || isPending || !showSave}>
          {isPending ? COPY.saving : COPY.save}
        </Button>
      </PermissionTooltip>
    </div>
  );
}
