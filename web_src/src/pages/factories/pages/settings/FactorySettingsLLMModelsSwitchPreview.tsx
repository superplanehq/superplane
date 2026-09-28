import { Check } from "lucide-react";
import { useEffect, useState } from "react";

import superplaneLogo from "@/assets/superplane.svg";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ModelAllowlistEditor } from "@/pages/organization/settings/ModelAllowlistEditor";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { byokProviderProductName, ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";

export type LLMModelsSwitchProvider = "anthropic" | "openai" | "openrouter";

export type LLMModelsSwitchTarget = "hosted" | LLMModelsSwitchProvider;

export type LLMModelsSwitchDialog = {
  target: LLMModelsSwitchTarget;
  step: "warn" | "key";
} | null;

const PROVIDERS: Array<{ id: LLMModelsSwitchProvider; integrationName: string }> = [
  { id: "anthropic", integrationName: "claude" },
  { id: "openai", integrationName: "openai" },
  { id: "openrouter", integrationName: "openrouter" },
];

const HOSTED_MODELS = ["anthropic/claude-sonnet-4-6", "openai/gpt-5"];

const PROVIDER_MODELS: Record<LLMModelsSwitchProvider, string[]> = {
  anthropic: ["claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"],
  openai: ["gpt-5", "gpt-4.1"],
  openrouter: ["anthropic/claude-sonnet-4-6", "openai/gpt-5"],
};

export interface FactorySettingsLLMModelsSwitchPreviewProps {
  /** Saved source. `hosted` is the SuperPlane agent. */
  source: "hosted" | LLMModelsSwitchProvider;
  dialog: LLMModelsSwitchDialog;
  switchedNotice: boolean;
  onChoose: (target: LLMModelsSwitchTarget) => void;
  onCancel: () => void;
  onContinueToKey: () => void;
  onBackToWarning: () => void;
  onSaveKey: (apiKey: string) => void;
  onSwitchToHosted: () => void;
}

export function FactorySettingsLLMModelsSwitchPreview({
  source,
  dialog,
  switchedNotice,
  onChoose,
  onCancel,
  onContinueToKey,
  onBackToWarning,
  onSaveKey,
  onSwitchToHosted,
}: FactorySettingsLLMModelsSwitchPreviewProps) {
  const hosted = source === "hosted";

  return (
    <FactorySettingsPageFrame title={COPY.pageTitle} subtitle={COPY.pageSubtitle}>
      <div data-testid="llm-models-switch-preview" className="flex flex-col gap-5">
        <FactorySettingsCard
          title={COPY.modelsTitle}
          data-testid="llm-models-list"
          action={
            <Badge variant="outline" data-testid="llm-models-source-badge">
              {hosted ? COPY.hostedBadge : COPY.ownKeyBadge}
            </Badge>
          }
        >
          {hosted ? (
            <HostedSource switchedNotice={switchedNotice} onChoose={onChoose} />
          ) : (
            <OwnKeySource provider={source} switchedNotice={switchedNotice} onChoose={onChoose} />
          )}
        </FactorySettingsCard>
      </div>
      <SwitchDialog
        source={source}
        dialog={dialog}
        onCancel={onCancel}
        onContinueToKey={onContinueToKey}
        onBackToWarning={onBackToWarning}
        onSaveKey={onSaveKey}
        onSwitchToHosted={onSwitchToHosted}
      />
    </FactorySettingsPageFrame>
  );
}

function HostedSource({
  switchedNotice,
  onChoose,
}: {
  switchedNotice: boolean;
  onChoose: (target: LLMModelsSwitchTarget) => void;
}) {
  return (
    <div className="space-y-5">
      <div className="space-y-3" data-testid="llm-models-hosted">
        {switchedNotice ? (
          <p className="text-xs text-muted-foreground" data-testid="llm-models-switched-notice">
            Automations in this workspace now use the SuperPlane agent.
          </p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          Agents in this workspace use SuperPlane models. SuperPlane bills these runs.
        </p>
        <ul className="max-h-56 space-y-2 overflow-auto">
          {HOSTED_MODELS.map((model) => (
            <li key={model} className="flex items-center gap-2 text-[13px]">
              <Check className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
              <span className="font-mono text-xs">{model}</span>
            </li>
          ))}
        </ul>
      </div>
      <div className="space-y-3 border-t border-border pt-4" data-testid="llm-models-provider-choices">
        <div>
          <h3 className="text-[13px] font-medium text-foreground">Use your own key</h3>
          <p className="mt-1 text-xs text-muted-foreground">Connect a provider key. The provider bills these runs.</p>
        </div>
        <ul className="space-y-2">
          {PROVIDERS.map((provider) => (
            <li
              key={provider.id}
              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
              data-testid={`llm-models-connect-${provider.id}`}
            >
              <span className="flex min-w-0 items-center gap-2 text-[13px]">
                <IntegrationIcon integrationName={provider.integrationName} className="size-4" size={16} />
                {byokProviderProductName(provider.id)}
              </span>
              <Button type="button" variant="outline" size="sm" onClick={() => onChoose(provider.id)}>
                Connect
              </Button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function OwnKeySource({
  provider,
  switchedNotice,
  onChoose,
}: {
  provider: LLMModelsSwitchProvider;
  switchedNotice: boolean;
  onChoose: (target: LLMModelsSwitchTarget) => void;
}) {
  const product = byokProviderProductName(provider);
  const others = PROVIDERS.filter((item) => item.id !== provider);
  return (
    <div className="space-y-5" data-testid={`llm-models-provider-${provider}`}>
      <div className="space-y-3">
        {switchedNotice ? (
          <p className="text-xs text-muted-foreground" data-testid="llm-models-switched-notice">
            Automations in this workspace now use the {product} agent.
          </p>
        ) : null}
        <div className="rounded-md border border-border bg-muted/40 px-3 py-2">
          <p className="text-[13px] font-medium text-foreground">Your {product} key</p>
          <p className="text-[12px] text-muted-foreground">{COPY.ownKeyHelper}</p>
        </div>
        <ProviderModelChecklist provider={provider} />
      </div>
      <div className="space-y-3 border-t border-border pt-4" data-testid="llm-models-change-source">
        <div>
          <h3 className="text-[13px] font-medium text-foreground">Change model source</h3>
          <p className="mt-1 text-xs text-muted-foreground">
            Switching replaces the {product} agent in every automation in this workspace.
          </p>
        </div>
        <ul className="space-y-2">
          <li
            className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
            data-testid="llm-models-switch-hosted"
          >
            <span className="flex min-w-0 items-center gap-2 text-[13px]">
              <SuperPlaneMark />
              SuperPlane
            </span>
            <Button type="button" variant="outline" size="sm" onClick={() => onChoose("hosted")}>
              Use SuperPlane
            </Button>
          </li>
          {others.map((item) => (
            <li
              key={item.id}
              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
              data-testid={`llm-models-connect-${item.id}`}
            >
              <span className="flex min-w-0 items-center gap-2 text-[13px]">
                <IntegrationIcon integrationName={item.integrationName} className="size-4" size={16} />
                {byokProviderProductName(item.id)}
              </span>
              <Button type="button" variant="outline" size="sm" onClick={() => onChoose(item.id)}>
                Connect
              </Button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function SuperPlaneMark() {
  return <img src={superplaneLogo} alt="" className="size-4 shrink-0 object-contain dark:brightness-0 dark:invert" />;
}

function ProviderModelChecklist({ provider }: { provider: LLMModelsSwitchProvider }) {
  const modelIds = PROVIDER_MODELS[provider];
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(modelIds);
  const [dirty, setDirty] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setQuery("");
    setSelected(modelIds);
    setDirty(false);
    setSaved(false);
  }, [provider, modelIds]);

  return (
    <div className="space-y-3">
      <ModelAllowlistEditor
        modelIds={modelIds}
        selected={selected}
        query={query}
        onQueryChange={setQuery}
        onToggle={(model, checked) => {
          setDirty(true);
          setSaved(false);
          setSelected((current) => (checked ? [...current, model] : current.filter((id) => id !== model)));
        }}
        disabled={false}
        searchLabel={`Search ${byokProviderProductName(provider)} models`}
        showCount
      />
      <div className="flex items-center gap-3">
        <Button
          type="button"
          disabled={!dirty || saved}
          onClick={() => {
            setDirty(false);
            setSaved(true);
          }}
        >
          {COPY.save}
        </Button>
        {saved ? <p className="text-xs text-muted-foreground">{COPY.saveSuccess}</p> : null}
      </div>
    </div>
  );
}

function sourceLabel(source: "hosted" | LLMModelsSwitchProvider): string {
  return source === "hosted" ? "SuperPlane" : byokProviderProductName(source);
}

export function SwitchDialog({
  source,
  dialog,
  onCancel,
  onContinueToKey,
  onBackToWarning,
  onSaveKey,
  onSwitchToHosted,
}: {
  source: "hosted" | LLMModelsSwitchProvider;
  dialog: LLMModelsSwitchDialog;
  onCancel: () => void;
  onContinueToKey: () => void;
  onBackToWarning: () => void;
  onSaveKey: (apiKey: string) => void;
  onSwitchToHosted: () => void;
}) {
  const [apiKey, setApiKey] = useState("");
  const current = sourceLabel(source);
  const next = dialog ? sourceLabel(dialog.target) : "";
  const open = dialog !== null;

  useEffect(() => {
    if (dialog?.step !== "key") {
      setApiKey("");
    }
  }, [dialog?.target, dialog?.step]);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onCancel();
        }
      }}
    >
      <DialogContent data-testid="llm-models-switch-dialog">
        {dialog?.step === "warn" ? (
          <>
            <DialogHeader>
              <DialogTitle>Switch automations to {next}?</DialogTitle>
              <DialogDescription>
                This replaces the {current} agent in every automation in this workspace with the {next} agent.{" "}
                {dialog.target === "hosted" ? "SuperPlane bills these runs." : `${next} bills these runs.`}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={onCancel}>
                Keep {current}
              </Button>
              {dialog.target === "hosted" ? (
                <Button type="button" onClick={onSwitchToHosted}>
                  Switch to SuperPlane
                </Button>
              ) : (
                <Button type="button" onClick={onContinueToKey}>
                  Switch to {next}
                </Button>
              )}
            </DialogFooter>
          </>
        ) : null}
        {dialog?.step === "key" ? (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              if (apiKey.trim() === "") {
                return;
              }
              onSaveKey(apiKey.trim());
              setApiKey("");
            }}
          >
            <DialogHeader>
              <DialogTitle>Your {next} key</DialogTitle>
              <DialogDescription>SuperPlane stores this key for agents in this workspace.</DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <Label htmlFor="llm-models-switch-api-key">{next} API key</Label>
              <Input
                id="llm-models-switch-api-key"
                data-testid="llm-models-switch-api-key"
                type="password"
                autoComplete="off"
                value={apiKey}
                onChange={(event) => setApiKey(event.target.value)}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={onBackToWarning}>
                Back
              </Button>
              <Button type="submit" disabled={apiKey.trim() === ""}>
                Save and switch
              </Button>
            </DialogFooter>
          </form>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
