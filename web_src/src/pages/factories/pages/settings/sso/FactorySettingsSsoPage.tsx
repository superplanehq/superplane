import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { SsoSettingsView } from "./SsoSettingsView";
import { useSsoSettingsScreen } from "./useSsoSettingsScreen";

export function SsoSettingsScreen({ organizationId }: { organizationId: string }) {
  const screen = useSsoSettingsScreen(organizationId);
  if (screen.loading) {
    return <p>Loading Single Sign-On.</p>;
  }
  return <SsoSettingsView key={screen.model.connection?.id ?? "new"} model={screen.model} actions={screen.actions} />;
}

export function FactorySettingsSsoPage() {
  const { organizationId } = useFactorySettingsLayout();
  return (
    <FactorySettingsPageFrame title="Single Sign-On" subtitle="Connect one identity provider for this organization.">
      <SsoSettingsScreen organizationId={organizationId} />
    </FactorySettingsPageFrame>
  );
}
