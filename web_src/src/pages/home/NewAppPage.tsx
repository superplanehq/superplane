import { RequirePermission } from "@/components/PermissionGate";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { HomePageShell } from "./HomePageShell";
import { ZeroStatePage } from "./ZeroStatePage";
import { useNewAppFolderContext } from "./useNewAppFolderContext";

export function NewAppPage() {
  const { folder, folderContextPending } = useNewAppFolderContext();
  const title = folder ? `Create New App in ${folder.title} Folder` : "Create New App";
  usePageTitle([title]);
  useReportPageReady(true);

  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <ZeroStatePage folder={folder} folderContextPending={folderContextPending} title={title} />
      </HomePageShell>
    </RequirePermission>
  );
}
