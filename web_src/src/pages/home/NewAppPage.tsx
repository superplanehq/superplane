import { RequirePermission } from "@/components/PermissionGate";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { FreshOrgLanding } from "./FreshOrgLanding";
import { HomePageShell } from "./HomePageShell";
import { RequireClassicAppsSurface } from "./RequireClassicAppsSurface";

export function NewAppPage() {
  return (
    <RequireClassicAppsSurface>
      <ClassicNewAppPage />
    </RequireClassicAppsSurface>
  );
}

function ClassicNewAppPage() {
  const title = "Create a new app";
  usePageTitle([title]);
  useReportPageReady(true);

  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <FreshOrgLanding title={title} />
      </HomePageShell>
    </RequirePermission>
  );
}
