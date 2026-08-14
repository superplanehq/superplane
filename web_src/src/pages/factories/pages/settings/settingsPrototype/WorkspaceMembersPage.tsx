import { Members } from "@/pages/organization/settings/Members";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";

export function WorkspaceMembersPage() {
  const { organizationId } = useFactorySettingsLayout();

  return (
    <WorkspaceSettingsSection
      title="Members"
      description="Invite people to this workspace and assign roles. Members can create work orders and review agent results."
    >
      <Members organizationId={organizationId} />
    </WorkspaceSettingsSection>
  );
}
