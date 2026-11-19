import type { ReactNode } from "react";

import { Badge } from "@/components/Badge/badge";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { DangerCard, PolicyCard } from "./SsoPolicySection";
import { ProvisioningCard, ScimCard } from "./SsoProvisioningSection";
import { ConfirmDialog, RequiredDialog, TokenDialog } from "./SsoSettingsDialogs";
import {
  ActivationCard,
  DomainCard,
  ProviderCard,
  ProviderUrlsCard,
  ProviderValuesCard,
  SetupCard,
} from "./SsoSettingsSections";
import { noopSsoActions, statusLabel, type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";
import { useSsoSettingsState } from "./useSsoSettingsState";

export type { SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

function statusBadge(status: string | undefined) {
  const label = statusLabel(status);
  const color = status === "STATUS_ACTIVE" ? "green" : status === "STATUS_DISABLED" ? "zinc" : "gray";
  return <Badge color={color}>{label}</Badge>;
}

const title = "Single Sign-On";
const subtitle = "Members sign in to this organization through your identity provider.";

export function SsoSettingsFrame({ actions, children }: { actions?: ReactNode; children: ReactNode }) {
  return (
    <FactorySettingsPageFrame title={title} subtitle={subtitle} actions={actions}>
      {children}
    </FactorySettingsPageFrame>
  );
}

export function SsoSettingsView({
  model,
  actions = noopSsoActions,
}: {
  model: SsoSettingsModel;
  actions?: SsoSettingsActions;
}) {
  const state = useSsoSettingsState(model, actions);
  if (!model.licensed && !state.exists) {
    return (
      <SsoSettingsFrame>
        <FactorySettingsCard data-testid="sso-locked">
          <Badge>Enterprise</Badge>
          <p className="mt-3 text-[13px] text-foreground">Single Sign-On needs a SuperPlane Enterprise license.</p>
        </FactorySettingsCard>
      </SsoSettingsFrame>
    );
  }

  const active = model.connection?.status === "STATUS_ACTIVE";
  return (
    <SsoSettingsFrame actions={state.exists ? statusBadge(model.connection?.status) : undefined}>
      <div data-testid="sso-settings" className="flex flex-col gap-5">
        {active ? null : <SetupCard draft={state.draft} />}
        <DomainCard model={model} actions={actions} draft={state.draft} />
        <ProviderCard model={model} draft={state.draft} />
        {state.exists ? <ProviderUrlsCard model={model} draft={state.draft} /> : null}
        {state.exists ? <ProviderValuesCard model={model} draft={state.draft} /> : null}
        <ActivationCard model={model} actions={actions} draft={state.draft} />
        <ProvisioningCard model={model} draft={state.draft} />
        {state.exists ? <ScimCard model={model} actions={actions} draft={state.draft} /> : null}
        <PolicyCard model={model} draft={state.draft} />
        {state.exists ? <DangerCard actions={actions} draft={state.draft} /> : null}
      </div>
      <ConfirmDialog confirm={state.confirm} onClose={() => state.setConfirm(null)} />
      <RequiredDialog
        open={state.requiredConfirm}
        onOpenChange={state.setRequiredConfirm}
        onConfirm={() => {
          state.draft.setEnforcement("ENFORCEMENT_REQUIRED");
          state.setRequiredConfirm(false);
        }}
      />
      <TokenDialog secret={model.revealedToken} onDismiss={actions.onDismissToken} />
    </SsoSettingsFrame>
  );
}
