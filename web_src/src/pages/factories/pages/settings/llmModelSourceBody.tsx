import { ArrowUpRight } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { HOSTED_LLM_PROVIDERS } from "@/lib/hostedLLMModels";
import { useFactoryLLMModels, type useBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";

import type { LLMModelsSwitchProvider } from "./FactorySettingsLLMModelsSwitchPreview";
import { byokProviderProductName, ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";
import type { WorkspaceModelSource } from "./workspaceModelSource";
import { WorkspaceModelPolicyEditor } from "./WorkspaceModelPolicyEditor";

type BYOKQuery = ReturnType<typeof useBYOKLLMModels>;

type ModelSourceBodyProps = {
  source: WorkspaceModelSource;
  providers: LLMModelsSwitchProvider[];
  currentProvider: LLMModelsSwitchProvider | null;
  organizationId: string;
  factoryId: string;
  byokQueries: Record<string, BYOKQuery>;
  canUpdate: boolean;
  integrationsHref: string;
};

export function ModelSourceBody(props: ModelSourceBodyProps) {
  const { source, organizationId, factoryId, canUpdate } = props;
  const editorProps = { organizationId, factoryId, canUpdate };
  if (source === "subscription")
    return (
      <div className="space-y-5" data-testid="llm-models-codex-subscription">
        <p className="text-xs text-muted-foreground">{COPY.subscriptionHelper}</p>
        <WorkspaceModelPolicyEditor
          {...editorProps}
          provider="openai"
          fundingSource="subscription"
          label="ChatGPT subscription"
        />
      </div>
    );

  if (source === "hosted")
    return (
      <div className="space-y-5" data-testid="llm-models-hosted">
        <HostedModelPolicies {...editorProps} />
      </div>
    );

  return <OwnKeyModelSource {...props} />;
}

function OwnKeyModelSource({
  currentProvider,
  providers,
  byokQueries,
  integrationsHref,
  canUpdate,
  organizationId,
  factoryId,
}: ModelSourceBodyProps) {
  const editorProps = { organizationId, factoryId, canUpdate };
  const providerQuery = currentProvider ? byokQueries[currentProvider] : undefined;
  const integrationId = providerQuery?.data?.integrationId;
  const integrationHref = integrationId ? `${integrationsHref}/${integrationId}` : integrationsHref;
  return (
    <div className="space-y-5" data-testid={`llm-models-provider-${currentProvider ?? "none"}`}>
      <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <span className="text-[13px] font-medium">
            {currentProvider ? byokProviderProductName(currentProvider) : "Your API key"}
          </span>
          <p role="status" className="text-xs text-muted-foreground">
            {providerConnectionStatus(providerQuery)}
          </p>
        </div>
        <Button asChild variant="outline" size="sm">
          <Link to={integrationHref}>
            Open integration
            <ArrowUpRight className="size-3.5" aria-hidden />
          </Link>
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{COPY.ownKeyHelper}</p>
      {currentProvider && providers.includes(currentProvider) ? (
        <WorkspaceModelPolicyEditor
          key={currentProvider}
          {...editorProps}
          provider={currentProvider}
          fundingSource="byok"
          label={byokProviderProductName(currentProvider)}
        />
      ) : (
        <p role="status" className="text-[13px] text-muted-foreground">
          {providers.length === 0
            ? "Your API key is not available for this workspace."
            : "Select a provider to connect a key and choose models."}
        </p>
      )}
    </div>
  );
}

function HostedModelPolicies(props: { organizationId: string; factoryId: string; canUpdate: boolean }) {
  const anthropic = useFactoryLLMModels(props.organizationId, props.factoryId, "anthropic", "hosted", true);
  const openai = useFactoryLLMModels(props.organizationId, props.factoryId, "openai", "hosted", true);
  const openrouter = useFactoryLLMModels(props.organizationId, props.factoryId, "openrouter", "hosted", true);
  const queries = [anthropic, openai, openrouter];
  const empty = queries.every(
    (query) =>
      !query.isLoading &&
      !query.isError &&
      (query.data?.parent?.length ?? 0) === 0 &&
      (query.data?.configuredModels?.length ?? 0) === 0,
  );
  if (empty)
    return (
      <p role="status" className="text-[13px] text-muted-foreground">
        No SuperPlane models are available. Ask an organization admin to make models available.
      </p>
    );
  return (
    <>
      {HOSTED_LLM_PROVIDERS.map((provider) => (
        <WorkspaceModelPolicyEditor
          key={provider}
          {...props}
          provider={provider}
          fundingSource="hosted"
          label={provider === "anthropic" ? "Anthropic" : byokProviderProductName(provider)}
          hideEmpty
        />
      ))}
    </>
  );
}

function providerConnectionStatus(query: BYOKQuery | undefined) {
  if (query?.isLoading) return "Checking connection...";
  if (query?.isError) return "Connection unavailable";
  return query?.data?.connected ? "Connected" : "Not connected";
}
