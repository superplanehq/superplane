import { Button } from "@/components/ui/button";
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
  const disabled = [
    !canAct("factories", "read"),
    isLoading,
    codex.query.isLoading,
    codex.query.data?.revocationPending,
    !offered,
  ].some(Boolean);
  return (
    <FactorySettingsCard
      title="Your ChatGPT subscription"
      description="Only your tasks use this connection in this workspace."
    >
      {features.isLoading || codex.query.isLoading ? <p role="status">Loading ChatGPT connection...</p> : null}
      {codex.query.data?.revocationPending ? (
        <p role="status">Stopping your previous tasks. You can reconnect when cleanup finishes.</p>
      ) : null}
      <FirstRunCodexSubscription
        selected
        disabled={disabled}
        connection={codex.query.data}
        starting={codex.start.isPending}
        canceling={codex.cancel.isPending}
        onSelect={() => {}}
        onStart={() => codex.start.mutate()}
        onCancel={() => codex.cancel.mutate()}
      />
      {codex.query.data?.id && codex.query.data.status !== "disconnected" ? (
        <Button
          type="button"
          variant="outline"
          disabled={codex.disconnect.isPending || !canAct("factories", "read")}
          onClick={() => codex.disconnect.mutate()}
        >
          Disconnect ChatGPT
        </Button>
      ) : null}
    </FactorySettingsCard>
  );
}
