import { RequirePermission } from "@/components/PermissionGate";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";

import { FreshOrgLanding } from "../FreshOrgLanding";
import { HomePageShell } from "../HomePageShell";
import { useNewAppFolderContext } from "../useNewAppFolderContext";

/**
 * Storybook-only `/apps/new` surface for the factory-first landing prototype.
 * Production `NewAppPage` continues to render `ZeroStatePage`.
 */
export function PrototypeNewAppPage() {
  const { folder, folderContextPending } = useNewAppFolderContext();
  const title = folder ? `Create New App in ${folder.title} Folder` : "Create a new app";
  usePageTitle([title]);
  useReportPageReady(true);

  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <FreshOrgLanding folder={folder} folderContextPending={folderContextPending} title={title} />
      </HomePageShell>
    </RequirePermission>
  );
}
