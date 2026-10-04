import type { ReactNode } from "react";
import { useParams } from "react-router";

import { FactoriesLayout } from "../layout/FactoriesLayout";
import { LinesPage } from "../pages/LinesPage";
import { WorkOrderDetailPage } from "../pages/WorkOrderDetailPage";
import { MobileBoardPage } from "./MobileBoardPage";
import { MobileFactoriesLayout } from "./MobileFactoriesLayout";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";
import { useMobileFactoryShell } from "./useMobileFactoryShell";

/**
 * Route-level switches between the desktop workspace shell and the mobile
 * proof-of-concept shell. Every switch keeps the desktop component as the
 * fallback, so nothing changes until the organization turns on the
 * `mobile_factory_board` feature and the viewport is phone-width.
 */
export function FactoryWorkspaceLayoutSwitch({ children }: { children?: ReactNode }) {
  const { organizationId } = useParams<{ organizationId: string }>();
  if (useMobileFactoryShell(organizationId)) {
    return <MobileFactoriesLayout>{children}</MobileFactoriesLayout>;
  }
  return <FactoriesLayout>{children}</FactoriesLayout>;
}

export function LineBoardRouteSwitch() {
  const { organizationId } = useParams<{ organizationId: string }>();
  return useMobileFactoryShell(organizationId) ? <MobileBoardPage /> : <LinesPage />;
}

export function WorkOrderDetailRouteSwitch() {
  const { organizationId } = useParams<{ organizationId: string }>();
  return useMobileFactoryShell(organizationId) ? <MobileTaskDetailPage /> : <WorkOrderDetailPage />;
}
