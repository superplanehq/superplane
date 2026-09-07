import type { FactoriesFactory } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganization } from "@/hooks/useOrganizationData";
import { useNavigate, useParams } from "react-router";
import { firstFactoryLineId, newFactoryPath } from "../lib/factoryPagePaths";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";
import { SidebarUserMenu } from "./SidebarUserMenu";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

interface FactoriesSidebarProps {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
}

/**
 * Icon rail shared by the workspace shell and workspace settings, so the
 * board, Velocity, and settings stay one click away on both screens.
 */
export function FactoriesSidebar({ organizationId, factoryKey, factory, factories }: FactoriesSidebarProps) {
  const navigate = useNavigate();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationId);
  const { lineId: routeLineId } = useParams<{ lineId?: string }>();

  return (
    <aside
      className="sticky top-0 flex h-screen w-[var(--workspace-navigation-width)] shrink-0 flex-col items-center border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
      data-testid="factories-sidebar"
    >
      <WorkspaceSwitcher
        organizationId={organizationId}
        factory={factory}
        factories={factories}
        canCreateFactory={canAct("factories", "create")}
        permissionsLoading={permissionsLoading}
        onCreateFactory={() => navigate(newFactoryPath(organizationId))}
      />
      <FactoriesSidebarNav
        organizationId={organizationId}
        factoryKey={factoryKey}
        lineId={routeLineId ?? firstFactoryLineId(factory)}
        canOpenSettings={canAct("factories", "update")}
        permissionsLoading={permissionsLoading}
      />
      <div className="flex-1" />
      <SidebarUserMenu
        organizationId={organizationId}
        factoryKey={factoryKey}
        userName={account?.name ?? "You"}
        userAvatarUrl={account?.avatar_url}
        organizationName={organization?.metadata?.name || "Organization"}
      />
    </aside>
  );
}
