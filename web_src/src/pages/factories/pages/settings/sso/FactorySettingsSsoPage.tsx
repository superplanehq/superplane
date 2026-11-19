import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { SsoSettingsFrame, SsoSettingsView } from "./SsoSettingsView";
import { useSsoSettingsScreen } from "./useSsoSettingsScreen";

export function SsoSettingsScreen({ organizationId }: { organizationId: string }) {
  const screen = useSsoSettingsScreen(organizationId);
  if (screen.loading) {
    return (
      <SsoSettingsFrame>
        <p className="text-[13px] text-muted-foreground">Loading...</p>
      </SsoSettingsFrame>
    );
  }
  return <SsoSettingsView key={screen.model.connection?.id ?? "new"} model={screen.model} actions={screen.actions} />;
}

export function FactorySettingsSsoPage() {
  const { organizationId } = useFactorySettingsLayout();
  return <SsoSettingsScreen organizationId={organizationId} />;
}
