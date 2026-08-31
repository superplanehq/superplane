import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, CircleDollarSign, Cpu, KeyRound } from "lucide-react";

import {
  factoryAgentProviders,
  providerFor,
  type AgentProvider,
  type CredentialSource,
  type ProviderOption,
} from "./useFactoryAgentSelection";

export function AgentSummary({
  onboarding,
}: {
  onboarding: { agentProvider?: string; agentModel?: string; agentPlanningModel?: string } | undefined;
}) {
  const provider = providerFor(onboarding);
  const model = onboarding?.agentModel || "onboarding default";
  const planningModel = onboarding?.agentPlanningModel || model;
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-muted/20 p-4">
      <Cpu className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-[13px] font-medium">
          {hostedProviderLabel(provider)} · {model} coding · {planningModel} planning
        </p>
        <p className="mt-1 text-[12px] text-muted-foreground">
          Planning uses the review model. Implementation and Backlog use the coding model.
        </p>
      </div>
    </div>
  );
}

export function AgentResolutionPreview({
  source,
  provider,
  resolution,
}: {
  source: CredentialSource;
  provider: ProviderOption;
  resolution: { codingModel: string; planningModel: string };
}) {
  return (
    <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
      <p className="text-[13px] font-medium">Will use {provider.runner}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">
        {source === "hosted" ? "SuperPlane-hosted credit" : `${hostedProviderLabel(provider.provider)} key`} · coding{" "}
        {resolution.codingModel} · planning {resolution.planningModel}
      </p>
    </div>
  );
}

export function AgentCredentialSelection({
  source,
  provider,
  integrationId,
  readyIntegrations,
  onProviderChange,
  onIntegrationChange,
}: {
  source: CredentialSource;
  provider: AgentProvider;
  integrationId: string;
  readyIntegrations: Array<{ metadata?: { id?: string; name?: string } }>;
  onProviderChange: (provider: AgentProvider) => void;
  onIntegrationChange: (integrationId: string) => void;
}) {
  if (source === "integration") {
    return (
      <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
        <Label htmlFor="factory-agent-integration" className="text-[13px] font-medium">
          {hostedProviderLabel(provider)} key
        </Label>
        {readyIntegrations.length > 0 ? (
          <Select value={integrationId} onValueChange={onIntegrationChange}>
            <SelectTrigger id="factory-agent-integration" className="mt-2 h-9">
              <SelectValue placeholder={`Select a ${hostedProviderLabel(provider)} key`} />
            </SelectTrigger>
            <SelectContent>
              {readyIntegrations.map((integration) => (
                <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ""}>
                  {integration.metadata?.name || integration.metadata?.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Text className="mt-2 text-[13px] text-muted-foreground">
            Connect a {hostedProviderLabel(provider)} key to use this provider.
          </Text>
        )}
      </div>
    );
  }

  return (
    <div className="mt-4 rounded-lg border border-border bg-muted/20 p-4">
      <Label htmlFor="factory-agent-hosted-provider" className="text-[13px] font-medium">
        Hosted provider
      </Label>
      <Select value={provider} onValueChange={(value) => onProviderChange(value as AgentProvider)}>
        <SelectTrigger id="factory-agent-hosted-provider" className="mt-2 h-9">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {factoryAgentProviders.map((option) => (
            <SelectItem key={option.provider} value={option.provider}>
              {hostedProviderLabel(option.provider)} · {option.runner}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function HostedOption({
  active,
  provider,
  onSelect,
}: {
  active: boolean;
  provider: AgentProvider;
  onSelect: () => void;
}) {
  return (
    <button type="button" onClick={onSelect} className={optionClassName(active)} aria-pressed={active}>
      <CircleDollarSign className="mt-0.5 size-5 shrink-0" aria-hidden />
      <span className="min-w-0 flex-1 text-left">
        <span className="block text-[13px] font-medium">SuperPlane-hosted credit</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">
          Use hosted {hostedProviderLabel(provider)} models for this workspace.
        </span>
      </span>
      {active ? <Check className="size-4 shrink-0" aria-hidden /> : null}
    </button>
  );
}

export function ProviderCard({
  active,
  option,
  readyCount,
  onSelect,
  onConnect,
}: {
  active: boolean;
  option: ProviderOption;
  readyCount: number;
  onSelect: () => void;
  onConnect: () => void;
}) {
  return (
    <div className={optionClassName(active)}>
      <button
        type="button"
        onClick={onSelect}
        className="flex min-w-0 flex-1 items-start gap-3 text-left"
        aria-pressed={active}
      >
        <IntegrationIcon integrationName={option.integrationName} className="mt-0.5 size-5 shrink-0" size={20} />
        <span className="min-w-0 flex-1">
          <span className="block text-[13px] font-medium">{hostedProviderLabel(option.provider)}</span>
          <span className="mt-0.5 block text-[12px] text-muted-foreground">
            Use a {hostedProviderLabel(option.provider)} key with {option.runner}.
          </span>
        </span>
      </button>
      {readyCount > 0 ? (
        <span className="flex shrink-0 items-center gap-1.5 text-[12px] text-emerald-700 dark:text-emerald-300">
          {active ? <Check className="size-3.5" aria-hidden /> : <KeyRound className="size-3.5" aria-hidden />}
          {readyCount === 1 ? "Connected" : `${readyCount} connected`}
        </span>
      ) : (
        <Button type="button" variant="outline" size="sm" onClick={onConnect}>
          Connect
        </Button>
      )}
    </div>
  );
}

function optionClassName(active: boolean): string {
  return `flex items-start gap-3 rounded-lg border p-4 transition-colors ${
    active ? "border-foreground bg-accent/40" : "border-border bg-background hover:bg-accent/20"
  }`;
}
