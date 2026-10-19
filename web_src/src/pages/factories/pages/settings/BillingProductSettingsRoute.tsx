import type { ReactNode } from "react";
import { Navigate, useParams } from "react-router";

import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { factorySettingsSectionPath } from "@/pages/factories/lib/factoryPagePaths";

export function BillingProductSettingsRoute({ children }: { children: ReactNode }) {
  const billingProductEnabled = useBillingProductEnabled();
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  if (!billingProductEnabled) {
    return <Navigate to={factorySettingsSectionPath(organizationId, factoryKey, "organization", "general")} replace />;
  }
  return children;
}
