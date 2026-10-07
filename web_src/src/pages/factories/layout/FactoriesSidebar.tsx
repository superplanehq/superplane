import type { FactoriesFactory } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useNavigate, useParams } from "react-router";
import { firstFactoryLineId, newFactoryPath } from "../lib/factoryPagePaths";
import { organizationPlanLabel } from "../lib/hostedCreditEmpty";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";
import { SidebarUserMenu } from "./SidebarUserMenu";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

const factoriesSidebarClassName =
  "sticky top-0 flex h-screen w-[var(--workspace-navigation-width)] shrink-0 flex-col items-center border-r border-sidebar-border bg-sidebar text-sidebar-foreground";

interface FactoriesSidebarProps {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
}

/**
 * Icon rail shared by the workspace shell and workspace settings.
 * Board and Velocity stay on the rail. Workspace settings open from the switcher.
 */
export function FactoriesSidebar({ organizationId, factoryKey, factory, factories }: FactoriesSidebarProps) {
  const navigate = useNavigate();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationId);
  const { lineId: routeLineId } = useParams<{ lineId?: string }>();
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const planLabel = organizationPlanLabel({
    purchasedCreditCents: spend.data?.purchasedCreditCents,
    welcomeCreditExpiresAt: spend.data?.welcomeCreditExpiresAt,
    plan: billing.data?.plan,
    trialEndsAt: billing.data?.trialEndsAt,
  });

  return (
    <aside className={factoriesSidebarClassName} data-testid="factories-sidebar">
      <WorkspaceSwitcher
        organizationId={organizationId}
        factory={factory}
        factories={factories}
        canCreateFactory={canAct("factories", "create")}
        canOpenSettings={canAct("factories", "update")}
        permissionsLoading={permissionsLoading}
        onCreateFactory={() => navigate(newFactoryPath(organizationId))}
      />
      <FactoriesSidebarNav
        organizationId={organizationId}
        factoryKey={factoryKey}
        lineId={routeLineId ?? firstFactoryLineId(factory)}
      />
      <div className="flex-1" />
      <SidebarUserMenu
        organizationId={organizationId}
        factoryKey={factoryKey}
        userName={account?.name ?? "You"}
        userAvatarUrl={account?.avatar_url}
        organizationName={organization?.metadata?.name || "Organization"}
        planLabel={planLabel}
      />
    </aside>
  );
}

/**
 * Same rail as the member workspace. The initials open workspace info only.
 * Velocity stays hidden. A signed-in visitor still gets the user menu.
 */
export function PublicFactoriesSidebar({
  organizationId,
  factoryKey,
  lineId,
  workspaceName,
  account,
}: {
  organizationId: string;
  factoryKey: string;
  lineId: string;
  workspaceName: string;
  account: { name?: string; avatarUrl?: string | null } | null;
}) {
  const factory = { id: factoryKey, key: factoryKey, name: workspaceName } as FactoriesFactory;

  return (
    <aside className={factoriesSidebarClassName} data-testid="factories-sidebar">
      <WorkspaceSwitcher
        infoOnly
        organizationId={organizationId}
        factory={factory}
        factories={[factory]}
        canCreateFactory={false}
        canOpenSettings={false}
        permissionsLoading={false}
        onCreateFactory={() => undefined}
      />
      <FactoriesSidebarNav
        organizationId={organizationId}
        factoryKey={factoryKey}
        lineId={lineId}
        showVelocity={false}
      />
      <div className="flex-1" />
      {account ? (
        <SidebarUserMenu
          visitor
          organizationId={organizationId}
          userName={account.name?.trim() || "You"}
          userAvatarUrl={account.avatarUrl}
          organizationName={account.name?.trim() || "You"}
        />
      ) : null}
    </aside>
  );
}
