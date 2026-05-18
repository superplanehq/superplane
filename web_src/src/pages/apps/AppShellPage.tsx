import { useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { useApp } from "@/hooks/useAppData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { usePermissions } from "@/contexts/usePermissions";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Loader2, MoreVertical, Trash2 } from "lucide-react";
import { AppDashboardTab } from "./AppDashboardTab";
import { AppCanvasTab } from "./AppCanvasTab";
import { AppDocsTab } from "./AppDocsTab";
import { SyncIndicator } from "./SyncIndicator";
import { DeleteAppDialog } from "./DeleteAppDialog";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

type AppTab = "dashboard" | "canvas" | "docs";

export function AppShellPage() {
  const { organizationId = "", appId = "" } = useParams<{ organizationId: string; appId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const { canAct } = usePermissions();

  const activeTab = (searchParams.get("view") as AppTab) ?? "dashboard";
  const appQuery = useApp(appId);
  const app = appQuery.data;

  usePageTitle([app?.metadata?.displayName ?? "App"]);

  const [isDeleteOpen, setIsDeleteOpen] = useState(false);

  const canUpdate = canAct("apps", "update");
  const canDelete = canAct("apps", "delete");

  const handleTabChange = (tab: string) => {
    if (tab === "canvas") {
      // Canvas tab navigates to the full canvas page (handled in AppCanvasTab)
      setSearchParams({ view: "canvas" }, { replace: false });
    } else {
      setSearchParams({ view: tab }, { replace: false });
    }
  };

  if (appQuery.isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (appQuery.error || !app) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-2">
        <Heading level={3}>App not found</Heading>
        <Text className="text-muted-foreground">This app may have been deleted or you don't have access.</Text>
      </div>
    );
  }

  return (
    <div className="flex h-dvh flex-col overflow-hidden">
      {/* Page header */}
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center gap-3 shrink-0">
        <OrganizationMenuButton organizationId={organizationId} />
        <div className="h-4 w-px bg-slate-200" />
        <div className="flex flex-1 items-center gap-3 min-w-0">
          <div className="min-w-0">
            <span className="font-semibold text-sm text-gray-900 dark:text-gray-100 truncate">
              {app.metadata?.displayName}
            </span>
            {app.metadata?.description && (
              <span className="hidden md:inline ml-2 text-xs text-muted-foreground truncate">
                {app.metadata.description}
              </span>
            )}
          </div>
          {app.syncState && (
            <div className="hidden sm:flex items-center">
              <SyncIndicator app={app} canSync={canUpdate} />
            </div>
          )}
        </div>

        {/* App actions menu */}
        {canDelete && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <MoreVertical className="h-4 w-4" />
                <span className="sr-only">App actions</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setIsDeleteOpen(true)} className="text-red-600 focus:text-red-600">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete App
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </header>

      {/* Tab bar */}
      <div className="bg-white border-b border-slate-950/15 px-4 shrink-0">
        <Tabs value={activeTab} onValueChange={handleTabChange}>
          <TabsList className="h-auto bg-transparent p-0 gap-0 rounded-none">
            <TabsTrigger
              value="dashboard"
              className="rounded-none border-0 border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-4 py-2.5 text-sm"
            >
              Dashboard
            </TabsTrigger>
            <TabsTrigger
              value="canvas"
              className="rounded-none border-0 border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-4 py-2.5 text-sm"
            >
              Canvas
            </TabsTrigger>
            <TabsTrigger
              value="docs"
              className="rounded-none border-0 border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-4 py-2.5 text-sm"
            >
              Docs
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "dashboard" && <AppDashboardTab appId={appId} readOnly={!canUpdate} />}
        {activeTab === "canvas" && <AppCanvasTab />}
        {activeTab === "docs" && <AppDocsTab appId={appId} readOnly={!canUpdate} />}
      </div>

      {/* Delete dialog */}
      {isDeleteOpen && (
        <DeleteAppDialog app={app} isOpen={isDeleteOpen} onClose={() => setIsDeleteOpen(false)} redirectOnDelete />
      )}
    </div>
  );
}
