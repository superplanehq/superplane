import { Secrets } from "@/pages/organization/settings/Secrets";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";

export function WorkspaceSecretsPage() {
  const { organizationId } = useFactorySettingsLayout();

  return (
    <WorkspaceSettingsSection
      title="Secrets"
      description="Store API keys and other secret values for this workspace. Components and agents can read them during runs."
    >
      <Secrets organizationId={organizationId} />
    </WorkspaceSettingsSection>
  );
}
