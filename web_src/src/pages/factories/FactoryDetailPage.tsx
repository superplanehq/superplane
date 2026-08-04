import { Text } from "@/components/Text/text";
import { cn } from "@/lib/utils";
import { useParams } from "react-router-dom";
import { CreateFactoryAppDialog } from "./CreateFactoryAppDialog";
import { FactoryAppsSidebar } from "./FactoryAppsSidebar";
import { FactoryDetailHeader } from "./FactoryDetailHeader";
import { FactoryDetailWorkOrdersPanel } from "./FactoryDetailWorkOrdersPanel";
import { FactoryLinesSidebar } from "./FactoryLinesSidebar";
import { FactoryPageShell } from "./FactoryPageShell";
import {
  factoryDetailPanelClassName,
  factoryDetailSidebarClassName,
  factoryPageContentClassName,
} from "./lib/factoryPageStyles";
import { useFactoryDetailPage } from "./useFactoryDetailPage";

export function FactoryDetailPage() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();

  if (!organizationId || !factoryId) {
    return null;
  }

  return <FactoryDetailPageContent organizationId={organizationId} factoryId={factoryId} />;
}

function FactoryDetailPageContent({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  const page = useFactoryDetailPage(organizationId, factoryId);

  if (page.kind === "redirect") {
    return page.element;
  }

  const factoryLines = page.factory?.lines ?? [];

  return (
    <FactoryPageShell backHref={`/${organizationId}/factories`} backLabel="Factories">
      {page.factoryLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading factory…</Text>
        </div>
      ) : page.factory ? (
        <div className={cn(factoryPageContentClassName, "pb-10")}>
          <FactoryDetailHeader
            factory={page.factory}
            workOrdersCount={page.openWorkOrderCount}
            canCreate={page.canCreateWork}
            permissionsLoading={page.permissionsLoading}
            createHref={page.createWorkOrderHref}
          />

          <div className={cn(factoryDetailPanelClassName, "mt-8 grid w-full lg:grid-cols-[minmax(0,1fr)_320px]")}>
            <FactoryDetailWorkOrdersPanel
              openWorkOrderCount={page.openWorkOrderCount}
              ownerFilter={page.ownerFilter}
              statusFilter={page.statusFilter}
              onOwnerFilterChange={page.setOwnerFilter}
              onStatusFilterChange={page.setStatusFilter}
              ordersError={page.ordersError}
              isOrdersLoading={page.isOrdersLoading}
              filteredWorkOrders={page.filteredWorkOrders}
              factoryHref={page.factoryHref}
              organizationId={page.organizationId}
              factoryLines={factoryLines}
              canDispatch={page.canDispatch}
              permissionsLoading={page.permissionsLoading}
              isDispatching={page.isDispatching}
              onDispatch={page.handleDispatch}
            />

            <div className={factoryDetailSidebarClassName}>
              <FactoryAppsSidebar
                organizationId={page.organizationId}
                apps={page.factoryApps}
                isLoading={page.isAppsLoading}
                canCreate={page.canCreateApps}
                permissionsLoading={page.permissionsLoading}
                onCreateClick={() => page.setCreateAppOpen(true)}
              >
                <FactoryLinesSidebar
                  factoryHref={page.factoryHref}
                  lines={factoryLines}
                  canUpdate={page.canUpdateFactory}
                />
              </FactoryAppsSidebar>
            </div>
          </div>
        </div>
      ) : null}

      <CreateFactoryAppDialog
        open={page.createAppOpen}
        isSaving={page.isCreateAppSaving}
        onClose={() => page.setCreateAppOpen(false)}
        onCreate={page.handleCreateApp}
      />
    </FactoryPageShell>
  );
}
