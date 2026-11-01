import { useParams } from "react-router";

import { Members } from "@/pages/organization/settings/Members";
import {
  OrganizationFleets,
  OrganizationFleetDetails,
  OrganizationRunnerDetails,
} from "@/pages/organization/settings/fleets/OrganizationFleets";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";

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

export function FactoryOrganizationFleetsPage({ detail = "catalog" }: { detail?: "catalog" | "fleet" | "runner" }) {
  const { organizationId = "", factoryKey = "" } = useParams();
  const basePath = factorySettingsSectionPath(organizationId, factoryKey, "organization", "fleets");
  return (
    <FactorySettingsPageFrame title="Fleets" subtitle="Manage organization fleets and directly connected runners.">
      {detail === "catalog" ? (
        <OrganizationFleets key={organizationId} organizationId={organizationId} basePath={basePath} />
      ) : detail === "fleet" ? (
        <OrganizationFleetDetails organizationId={organizationId} basePath={basePath} />
      ) : (
        <OrganizationRunnerDetails organizationId={organizationId} basePath={basePath} />
      )}
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
