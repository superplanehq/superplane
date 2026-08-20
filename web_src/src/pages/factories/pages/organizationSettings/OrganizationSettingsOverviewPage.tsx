import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useParams } from "react-router";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsOverviewPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: organization } = useOrganization(organizationId || "");
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle(["General", organizationName]);

  return (
    <FactorySettingsPageFrame title="General" subtitle="See the organization name and basic details.">
      <FactorySettingsCard title="Organization" data-testid="organization-settings-overview">
        <dl className="space-y-1">
          <dt className="text-[12px] text-muted-foreground">Name</dt>
          <dd className="text-[13px] text-foreground" data-testid="organization-settings-overview-name">
            {organizationName}
          </dd>
        </dl>
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}
