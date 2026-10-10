import { Navigate } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factorySettingsPath } from "../lib/factoryPagePaths";
import { MobileMorePage } from "./MobileMorePage";
import type { ReactNode } from "react";

import { FactoriesLayout } from "../layout/FactoriesLayout";
import { LinesPage } from "../pages/LinesPage";
import { WorkOrderDetailPage } from "../pages/WorkOrderDetailPage";
import { MobileBoardPage } from "./MobileBoardPage";
import { MobileFactoriesLayout } from "./MobileFactoriesLayout";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";
import { useMobileFactoryShell } from "./useMobileFactoryShell";

/**
 * Route-level switches between the desktop workspace shell and the mobile
 * shell. Every switch keeps the desktop component as the fallback for
 * wider viewports; phone-width viewports render the mobile shell.
 */
export function FactoryWorkspaceLayoutSwitch({ children }: { children?: ReactNode }) {
  if (useMobileFactoryShell()) {
    return <MobileFactoriesLayout>{children}</MobileFactoriesLayout>;
  }
  return <FactoriesLayout>{children}</FactoriesLayout>;
}

export function LineBoardRouteSwitch() {
  return useMobileFactoryShell() ? <MobileBoardPage /> : <LinesPage />;
}

export function WorkOrderDetailRouteSwitch() {
  return useMobileFactoryShell() ? <MobileTaskDetailPage /> : <WorkOrderDetailPage />;
}

export function FactoryMoreRouteSwitch() {
  const mobile = useMobileFactoryShell();
  const { organizationId, routeSegment } = useFactoriesLayout();
  return mobile ? <MobileMorePage /> : <Navigate to={factorySettingsPath(organizationId, routeSegment)} replace />;
}
