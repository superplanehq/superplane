import { useParams } from "react-router";

import { APIKeyDetail } from "@/pages/organization/settings/ApiKeyDetail";
import { APIKeys } from "@/pages/organization/settings/ApiKeys";
import { Members } from "@/pages/organization/settings/Members";
import { SecretDetail } from "@/pages/organization/settings/SecretDetail";
import { Secrets } from "@/pages/organization/settings/Secrets";

import { FactorySettingsPageFrame } from "./FactorySettingsCard";

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
  const organizationId = useOrganizationId();
  return (
    <FactorySettingsPageFrame title="API keys" subtitle="Create and manage API keys for programmatic access.">
      <APIKeys organizationId={organizationId} />
    </FactorySettingsPageFrame>
  );
}

export function FactoryOrganizationApiKeyDetailPage() {
  const organizationId = useOrganizationId();
  return <APIKeyDetail organizationId={organizationId} />;
}

export function FactoryOrganizationSecretsPage() {
  const organizationId = useOrganizationId();
  return (
    <FactorySettingsPageFrame title="Secrets" subtitle="Store and manage organization secrets.">
      <Secrets organizationId={organizationId} />
    </FactorySettingsPageFrame>
  );
}

export function FactoryOrganizationSecretDetailPage() {
  const organizationId = useOrganizationId();
  return <SecretDetail organizationId={organizationId} />;
}
