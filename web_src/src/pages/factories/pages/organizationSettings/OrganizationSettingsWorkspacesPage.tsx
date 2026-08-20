import type { FactoriesFactory } from "@/api-client";
import { useFactories } from "@/hooks/useFactoryData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useParams } from "react-router";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsWorkspacesPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: organization } = useOrganization(organizationId || "");
  const { data: factories = [], isLoading, error } = useFactories(organizationId || "");
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle(["Workspaces", organizationName]);

  return (
    <FactorySettingsPageFrame title="Workspaces" subtitle="See the workspaces in this organization.">
      <FactorySettingsCard data-testid="organization-settings-workspaces">
        <WorkspacesBody isLoading={isLoading} error={error} factories={factories} />
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}

function WorkspacesBody({
  isLoading,
  error,
  factories,
}: {
  isLoading: boolean;
  error: unknown;
  factories: FactoriesFactory[];
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading workspaces…</p>;
  }

  if (error) {
    return <p className="text-[13px] text-destructive">Failed to load workspaces.</p>;
  }

  if (factories.length === 0) {
    return <p className="text-[13px] text-muted-foreground">This organization has no workspaces.</p>;
  }

  return (
    <table className="w-full text-left" data-testid="organization-settings-workspaces-table">
      <thead>
        <tr className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
          <th className="pb-2 pr-4 font-medium">Name</th>
          <th className="pb-2 font-medium">Key</th>
        </tr>
      </thead>
      <tbody className="text-[13px] text-foreground">
        {factories.map((factory) => (
          <tr key={factory.id} data-testid={`organization-settings-workspace-${factory.id}`}>
            <td className="py-1.5 pr-4">{factory.name || "Workspace"}</td>
            <td className="py-1.5 text-muted-foreground">{factory.key || "None"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
