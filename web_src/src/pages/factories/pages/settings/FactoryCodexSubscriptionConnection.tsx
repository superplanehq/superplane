import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { FirstRunCodexSubscription } from "../onboarding/first-run/FirstRunCodexSubscription";
import type { useFactoryCodexConnection } from "../onboarding/useFactoryCodexConnection";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactoryCodexSubscriptionConnection({ codex }: { codex: ReturnType<typeof useFactoryCodexConnection> }) {
  const { canAct, isLoading } = usePermissions();
  const { organizationId } = useFactorySettingsLayout();
  const features = useExperimentalFeature(organizationId);
  const offered =
    !features.isLoading && features.has(FEATURE_CODEX_SUBSCRIPTIONS) && codex.query.data?.eligible === true;
  return (
    <FactorySettingsCard title="ChatGPT subscription">
      {features.isLoading || codex.query.isLoading ? <p role="status">Loading ChatGPT connection...</p> : null}
      <FirstRunCodexSubscription
        selected
        disabled={!canAct("factories", "update") || isLoading || codex.query.isLoading || !offered}
        connection={codex.query.data}
        starting={codex.start.isPending}
        canceling={codex.cancel.isPending}
        onSelect={() => {}}
        onStart={() => codex.start.mutate()}
        onCancel={() => codex.cancel.mutate()}
      />
    </FactorySettingsCard>
  );
}
