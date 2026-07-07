import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { HomePageShell } from "./HomePageShell";
import { ZeroStatePage } from "./ZeroStatePage";
import { useNewAppFolderContext } from "./useNewAppFolderContext";

export function NewAppPage() {
  const { folder } = useNewAppFolderContext();
  const title = folder ? `Create New App in ${folder.title} Folder` : "Create New App";
  usePageTitle([title]);
  useReportPageReady(true);

  return (
    <HomePageShell>
      <ZeroStatePage folder={folder} title={title} />
    </HomePageShell>
  );
}
