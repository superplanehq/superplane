import { Integrations } from "@/pages/organization/settings/Integrations";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";

export function WorkspaceIntegrationsPage() {
  const { organizationId } = useFactorySettingsLayout();

  return (
    <WorkspaceSettingsSection
      title="Integrations"
      description="Connect the tools this workspace can use. SuperPlane uses these connections for repositories, models, and notifications."
    >
      <div className="max-w-3xl">
        <Integrations organizationId={organizationId} />
      </div>
    </WorkspaceSettingsSection>
  );
}
