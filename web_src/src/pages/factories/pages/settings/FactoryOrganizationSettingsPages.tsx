import { useParams } from "react-router";

import { Members } from "@/pages/organization/settings/Members";

import { FactorySettingsApiKeysPage } from "./FactorySettingsApiKeysPage";
import { FactorySettingsPageFrame } from "./FactorySettingsCard";
import { FactorySettingsOrganizationLLMModelsPage } from "./FactorySettingsOrganizationLLMModelsPage";
import { FactorySettingsSecretsPage } from "./FactorySettingsSecretsPage";

function useOrganizationId() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  return organizationId;
}

export function FactoryOrganizationMembersPage() {
  const organizationId = useOrganizationId();
  return (
    <FactorySettingsPageFrame title="Members" subtitle="Invite people and manage organization access.">
      <Members organizationId={organizationId} />
    </FactorySettingsPageFrame>
  );
}

export function FactoryOrganizationApiKeysPage() {
  return <FactorySettingsApiKeysPage />;
}

export function FactoryOrganizationApiKeyDetailPage() {
  return <FactorySettingsApiKeysPage />;
}

export function FactoryOrganizationSecretsPage() {
  return <FactorySettingsSecretsPage />;
}

export function FactoryOrganizationSecretDetailPage() {
  return <FactorySettingsSecretsPage />;
}

export function FactoryOrganizationLLMModelsPage() {
  return <FactorySettingsOrganizationLLMModelsPage />;
}
