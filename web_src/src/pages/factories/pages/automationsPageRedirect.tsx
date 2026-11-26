import { Navigate } from "react-router";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { automationsPath, factoryHomePath } from "../lib/factoryPagePaths";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";

type AutomationsLegacyRedirectProps = {
  organizationId: string;
  factoryKey: string;
  factoryLoaded: boolean;
  legacyLineId?: string;
};

/** Redirect stale /automations/:lineId bookmarks once factory lines resolve. */
export function AutomationsLegacyRedirect({
  organizationId,
  factoryKey,
  factoryLoaded,
  legacyLineId,
}: AutomationsLegacyRedirectProps) {
  if (!factoryLoaded) {
    return (
      <>
        <WorkspacePageHeader className={factorySectionHeaderClassName} title="Automations" />
        <div className={factorySectionBodyClassName}>
          <p className="text-[13px] text-muted-foreground">Loading…</p>
        </div>
      </>
    );
  }
  if (legacyLineId) {
    return <Navigate to={factoryHomePath(organizationId, factoryKey)} replace />;
  }
  return <Navigate to={automationsPath(organizationId, factoryKey)} replace />;
}
