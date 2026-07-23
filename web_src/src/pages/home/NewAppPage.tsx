import { RequirePermission } from "@/components/PermissionGate";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { FreshOrgLanding } from "./FreshOrgLanding";
import { HomePageShell } from "./HomePageShell";
import { useNewAppFolderContext } from "./useNewAppFolderContext";

export function NewAppPage() {
  const { folder, folderContextPending } = useNewAppFolderContext();
  const title = folder ? `Create New App in ${folder.title} Folder` : "Create New App";
  usePageTitle([title]);
  useReportPageReady(true);

  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <FreshOrgLanding folder={folder} folderContextPending={folderContextPending} />
      </HomePageShell>
    </RequirePermission>
  );
}
