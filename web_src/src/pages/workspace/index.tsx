import { HomePageShell } from "@/pages/home/HomePageShell";

import { SoftwareFactoryHome } from "./SoftwareFactoryHome";
import type { CreateWorkRequest, FactoryTab, WorkspacePageData } from "./types";

interface WorkspacePageProps {
  data: WorkspacePageData;
  defaultTab?: FactoryTab;
  onCreateWork?: (request: CreateWorkRequest) => void;
}

export function WorkspacePage({ data, defaultTab, onCreateWork }: WorkspacePageProps) {
  return (
    <HomePageShell>
      <SoftwareFactoryHome data={data} defaultTab={defaultTab} onCreateWork={onCreateWork} />
    </HomePageShell>
  );
}
