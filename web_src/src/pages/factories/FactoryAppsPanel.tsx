import type { FactoryApp } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { LayoutGrid } from "lucide-react";
import { homeListCardClassName } from "../home/homePageStyles";

interface FactoryAppsPanelProps {
  organizationId: string;
  apps: FactoryApp[];
  isLoading: boolean;
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
}

export function FactoryAppsPanel({
  organizationId,
  apps,
  isLoading,
  canCreate,
  permissionsLoading,
  onCreateClick,
}: FactoryAppsPanelProps) {
  if (isLoading) {
    return <Text className="text-sm text-gray-500">Loading apps…</Text>;
  }

  if (apps.length === 0) {
    return (
      <div className="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-6 py-12 text-center dark:border-gray-700 dark:bg-gray-900">
        <LayoutGrid className="h-10 w-10 text-slate-400 dark:text-gray-500" aria-hidden />
        <p className="mt-4 text-base font-medium text-slate-900 dark:text-gray-100">No factory apps yet</p>
        <p className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
          Factory apps are canvases owned by this factory. Lines reference them as runApp steps.
        </p>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create apps."
        >
          <Button type="button" className="mt-6" onClick={onCreateClick} disabled={!canCreate}>
            <Icon name="plus" />
            Create app
          </Button>
        </PermissionTooltip>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Text className="text-sm text-gray-500 dark:text-gray-400">
          {apps.length} app{apps.length === 1 ? "" : "s"} owned by this factory
        </Text>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create apps."
        >
          <Button type="button" onClick={onCreateClick} disabled={!canCreate} data-testid="factory-apps-create-button">
            <Icon name="plus" />
            Create app
          </Button>
        </PermissionTooltip>
      </div>

      <ul className="grid gap-3 sm:grid-cols-2">
        {apps.map((app) => (
          <li key={app.id}>
            <Link
              href={appPath(organizationId, app.id ?? "")}
              className={cn(
                homeListCardClassName,
                "block p-4 no-underline hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
              )}
            >
              <div className="flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300">
                  <LayoutGrid className="h-5 w-5" aria-hidden />
                </div>
                <div className="min-w-0">
                  <p className="truncate font-medium text-slate-900 dark:text-gray-100">{app.name}</p>
                  {app.description ? (
                    <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">{app.description}</p>
                  ) : null}
                </div>
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
