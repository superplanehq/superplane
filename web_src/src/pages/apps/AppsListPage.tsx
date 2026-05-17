import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useApps } from "@/hooks/useAppData";
import { AppActionsMenu } from "./AppActionsMenu";
import { CreateAppDialog } from "./CreateAppDialog";
import { SyncIndicator } from "./SyncIndicator";
import type { AppsApp } from "@/lib/appsApi";
import { BoxSelect, Loader2, Plus, Search } from "lucide-react";

export function AppsListPage() {
  usePageTitle(["Apps"]);
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [searchQuery, setSearchQuery] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const { data: apps = [], isLoading, error } = useApps(organizationId);

  const canCreateApps = canAct("apps", "create");
  const canUpdateApps = canAct("apps", "update");
  const canDeleteApps = canAct("apps", "delete");

  const formatDate = (value?: string) => {
    if (!value) return "Unknown";
    return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  };

  const filteredApps = apps.filter((app) => {
    const q = searchQuery.toLowerCase();
    return (
      app.metadata?.displayName?.toLowerCase().includes(q) ||
      app.metadata?.description?.toLowerCase().includes(q) ||
      app.metadata?.slug?.toLowerCase().includes(q)
    );
  });

  const canCreate = canCreateApps || permissionsLoading;

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </header>

      <main className="w-full h-full flex flex-col flex-grow">
        <div className="bg-slate-100 w-full flex-grow">
          <div className="mx-auto w-full max-w-6xl p-8">
            {/* Page title */}
            <div className="mb-6">
              <Heading level={2} className="!text-2xl mb-1">
                Apps
              </Heading>
              <Text className="text-gray-800 dark:text-gray-400">
                All apps in your organization. Each app has a Dashboard, Canvas, and Documentation.
              </Text>
            </div>

            {/* Toolbar */}
            <div className="mb-6 flex w-full flex-col gap-3 sm:flex-row sm:items-center">
              <PermissionTooltip allowed={canCreate} message="You don't have permission to create apps.">
                {canCreate ? (
                  <Button onClick={() => setIsCreateOpen(true)} aria-label="Create new app">
                    <Plus className="h-4 w-4" />
                    New App
                  </Button>
                ) : (
                  <Button type="button" disabled>
                    <Plus className="h-4 w-4" />
                    New App
                  </Button>
                )}
              </PermissionTooltip>

              <div className="min-w-0 w-full sm:ml-auto sm:w-80">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
                  <Input
                    placeholder="Filter apps…"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-10"
                  />
                </div>
              </div>
            </div>

            {/* Content */}
            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : error ? (
              <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
                <Text>Failed to load apps. Please try again later.</Text>
              </div>
            ) : filteredApps.length === 0 && searchQuery ? (
              <AppsSearchEmptyState />
            ) : filteredApps.length === 0 ? (
              <AppsEmptyState canCreate={canCreate} onCreateClick={() => setIsCreateOpen(true)} />
            ) : (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                {filteredApps.map((app) => (
                  <AppCard
                    key={app.metadata?.id}
                    app={app}
                    organizationId={organizationId}
                    formatDate={formatDate}
                    canUpdateApps={canUpdateApps}
                    canDeleteApps={canDeleteApps}
                    permissionsLoading={permissionsLoading}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </main>

      <CreateAppDialog isOpen={isCreateOpen} onClose={() => setIsCreateOpen(false)} />
    </div>
  );
}

interface AppCardProps {
  app: AppsApp;
  organizationId: string;
  formatDate: (value?: string) => string;
  canUpdateApps: boolean;
  canDeleteApps: boolean;
  permissionsLoading: boolean;
}

function AppCard({ app, organizationId, formatDate, canUpdateApps, canDeleteApps, permissionsLoading }: AppCardProps) {
  const appId = app.metadata?.id ?? "";
  const appHref = `/${organizationId}/apps/${appId}`;

  return (
    <div className="relative bg-white dark:bg-gray-800 rounded-md outline outline-gray-950/15 hover:shadow-md transition-shadow cursor-pointer">
      <Link to={appHref} aria-label={`Open app ${app.metadata?.displayName}`} className="absolute inset-0 rounded-md" />

      <div className="p-4">
        <div className="flex items-start justify-between mb-2">
          <div className="min-w-0 flex-1">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100 text-sm truncate">
              {app.metadata?.displayName}
            </h3>
            {app.metadata?.slug && (
              <p className="text-xs font-mono text-muted-foreground truncate mt-0.5">{app.metadata.slug}</p>
            )}
          </div>
          <div className="relative z-10 ml-2 shrink-0">
            <AppActionsMenu
              app={app}
              organizationId={organizationId}
              canUpdateApps={canUpdateApps}
              canDeleteApps={canDeleteApps}
              permissionsLoading={permissionsLoading}
            />
          </div>
        </div>

        {app.metadata?.description && (
          <p className="text-xs text-muted-foreground mb-3 line-clamp-2">{app.metadata.description}</p>
        )}

        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            Created {formatDate(app.metadata?.createdAt)}
          </span>
          {app.syncState && (
            <div className="relative z-10">
              <SyncIndicator app={app} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function AppsSearchEmptyState() {
  return (
    <div className="text-center py-12">
      <BoxSelect className="mx-auto text-gray-400 mb-4" size={48} aria-hidden />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        No apps found
      </Heading>
      <Text className="text-gray-500 dark:text-gray-400">
        Nothing matches that filter, try another word or clear it.
      </Text>
    </div>
  );
}

function AppsEmptyState({ canCreate, onCreateClick }: { canCreate: boolean; onCreateClick: () => void }) {
  return (
    <div className="text-center py-12">
      <BoxSelect className="mx-auto text-gray-400 mb-4" size={48} aria-hidden />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        No apps yet
      </Heading>
      <Text className="text-gray-500 dark:text-gray-400 mb-6">
        Create your first App to get started with a Dashboard, Canvas, and Documentation.
      </Text>
      {canCreate && (
        <Button onClick={onCreateClick}>
          <Plus className="h-4 w-4" />
          New App
        </Button>
      )}
    </div>
  );
}
