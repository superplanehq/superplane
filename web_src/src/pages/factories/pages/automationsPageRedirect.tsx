import { Navigate } from "react-router";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { automationsPath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

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
        <WorkspacePageHeader title="Automations" />
        <div className={factoryContentBodyClassName}>
          <p className="text-[13px] text-muted-foreground">Loading…</p>
        </div>
      </>
    );
  }
  if (legacyLineId) {
    return <Navigate to={factoryLineDetailPath(organizationId, factoryKey, legacyLineId)} replace />;
  }
  return <Navigate to={automationsPath(organizationId, factoryKey)} replace />;
}
