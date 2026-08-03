import type { FactoriesFactory } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

interface FactoryDetailHeaderProps {
  factory: FactoriesFactory;
  workOrdersCount: number;
  canCreate: boolean;
  permissionsLoading: boolean;
  createHref: string;
}

export function FactoryDetailHeader({
  factory,
  workOrdersCount: _workOrdersCount,
  canCreate,
  permissionsLoading,
  createHref,
}: FactoryDetailHeaderProps) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0 flex-1">
        <h1 className="text-3xl font-semibold tracking-tight text-gray-900 dark:text-gray-100">{factory.name}</h1>
        {factory.description?.trim() ? (
          <p className="mt-3 max-w-3xl text-sm leading-relaxed text-gray-500 dark:text-gray-400">
            {factory.description.trim()}
          </p>
        ) : null}
      </div>

      <PermissionTooltip
        allowed={canCreate || permissionsLoading}
        message="You don't have permission to create work orders."
      >
        <Button
          type="button"
          asChild
          disabled={!canCreate}
          className={cn(appDarkModeClasses.primaryAction)}
          data-testid="work-order-list-create-button"
        >
          <Link href={createHref}>
            <Icon name="plus" />
            New Work Order
          </Link>
        </Button>
      </PermissionTooltip>
    </div>
  );
}
