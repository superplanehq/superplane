import type { FactoriesWorkOrder } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { BacklogCreatePopover } from "../pages/BacklogCreatePopover";
import { useBacklogCreateMenu } from "../pages/useBacklogCreateMenu";

export function MobileCreateTaskButton({
  className,
  onImported,
}: {
  className: string;
  onImported: (order: FactoriesWorkOrder) => void;
}) {
  const { organizationId, factoryId, openCreateWorkOrder } = useFactoriesLayout();
  const menu = useBacklogCreateMenu(organizationId, factoryId, (_orderId, order) => {
    if (order) onImported(order);
  });

  return (
    <BacklogCreatePopover
      canAdd
      variant="mobile"
      trigger={
        <Button className={className}>
          <Plus className="size-5" aria-hidden />
          Create task
        </Button>
      }
      sources={menu.sources}
      items={menu.items}
      query={menu.query}
      focusedIntakeId={menu.focusedIntakeId}
      onQueryChange={menu.setQuery}
      onFocusedIntakeChange={menu.setFocusedIntake}
      onCreateManually={openCreateWorkOrder}
      onImportItem={menu.importItem}
      isLoading={menu.isLoading}
      isLoadingMore={menu.isLoadingMore}
      hasMore={menu.hasMore}
      onLoadMore={menu.loadMore}
      errorMessage={menu.errorMessage}
    />
  );
}
