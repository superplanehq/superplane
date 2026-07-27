import { HomePageShell } from "@/pages/home/HomePageShell";

import { SoftwareFactoryHome } from "./SoftwareFactoryHome";
import type { CreateWorkRequest, WorkspacePageData } from "./types";

interface WorkspacePageProps {
  data: WorkspacePageData;
  onCreateWork?: (request: CreateWorkRequest) => void;
}

export function WorkspacePage({ data, onCreateWork }: WorkspacePageProps) {
  return (
    <HomePageShell>
      <SoftwareFactoryHome data={data} onCreateWork={onCreateWork} />
    </HomePageShell>
  );
}
