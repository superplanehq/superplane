import { Badge } from "@/components/Badge/badge";

import {
  ActivationSection,
  ConfirmDialog,
  DomainSection,
  PolicySection,
  ProviderSection,
  ProvisioningSection,
  RequiredDialog,
  TokenDialog,
} from "./SsoSettingsSections";
import { noopSsoActions, type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";
import { useSsoSettingsState } from "./useSsoSettingsState";

export type { SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

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
      <section data-testid="sso-locked">
        <Badge>Enterprise</Badge>
        <p>This feature needs a SuperPlane Enterprise license.</p>
      </section>
    );
  }

  return (
    <div className="space-y-8" data-testid="sso-settings">
      {model.error ? (
        <p role="alert" data-testid="sso-error">
          {model.error}
        </p>
      ) : null}
      <DomainSection model={model} actions={actions} draft={state.draft} />
      <ProviderSection model={model} actions={actions} draft={state.draft} />
      <ActivationSection model={model} actions={actions} draft={state.draft} />
      <ProvisioningSection model={model} actions={actions} draft={state.draft} />
      <PolicySection draft={state.draft} />
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
    </div>
  );
}
