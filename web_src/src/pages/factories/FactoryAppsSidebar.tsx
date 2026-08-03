import type { FactoryApp } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { ExternalLink } from "lucide-react";
import { factoryCountBadgeClassName, factorySidebarActionClassName } from "./factoryPageStyles";

interface FactoryAppsSidebarProps {
  organizationId: string;
  apps: FactoryApp[];
  isLoading: boolean;
  canCreate: boolean;
  permissionsLoading: boolean;
  onCreateClick: () => void;
  children?: React.ReactNode;
}

export function FactoryAppsSidebar({
  organizationId,
  apps,
  isLoading,
  canCreate,
  permissionsLoading,
  onCreateClick,
  children,
}: FactoryAppsSidebarProps) {
  return (
    <aside className="space-y-6" data-testid="factory-apps-sidebar">
      <div className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Apps</h2>
            <span className={factoryCountBadgeClassName}>{apps.length}</span>
          </div>
          <PermissionTooltip
            allowed={canCreate || permissionsLoading}
            message="You don't have permission to create apps."
          >
            <Button
              type="button"
              variant="ghost"
              size="xs"
              onClick={onCreateClick}
              disabled={!canCreate}
              className="rounded-md text-gray-700 dark:text-gray-300"
              data-testid="factory-apps-sidebar-create"
            >
              + New
            </Button>
          </PermissionTooltip>
        </div>

        {isLoading ? (
          <Text className="text-sm text-gray-500 dark:text-gray-400">Loading apps…</Text>
        ) : apps.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">No apps yet.</p>
        ) : (
          <ul className="space-y-2">
            {apps.map((app) => (
              <li key={app.id}>
                <Link
                  href={appPath(organizationId, app.id ?? "")}
                  className={cn(factorySidebarActionClassName, "no-underline")}
                  data-testid={`factory-app-sidebar-${app.id}`}
                >
                  {app.name}
                  <ExternalLink className="h-3.5 w-3.5 shrink-0 opacity-70" aria-hidden />
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>

      {children}
    </aside>
  );
}
