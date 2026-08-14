import { Heading } from "@/components/Heading/heading";
import { cn } from "@/lib/utils";
import { Navigate } from "react-router";
import { automationsPath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

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
        <header className={factoryContentHeaderClassName}>
          <div>
            <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
              Automations
            </Heading>
          </div>
        </header>
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
