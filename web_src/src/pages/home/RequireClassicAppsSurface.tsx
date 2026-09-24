import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { factoryListPath } from "@/pages/factories/lib/factoryPagePaths";
import type { ReactNode } from "react";
import { Navigate, useParams } from "react-router";

/**
 * Keeps the classic Apps home and /apps/new landing off-limits when
 * FEATURE_FACTORIES is on. /workspaces then picks last-accessed or first
 * workspace, or the empty onboarding card.
 */
export function RequireClassicAppsSurface({ children }: { children: ReactNode }) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { has, isLoading } = useExperimentalFeature(organizationId);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center" data-testid="classic-apps-surface-loading">
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  if (has(FEATURE_FACTORIES) && organizationId) {
    return <Navigate to={factoryListPath(organizationId)} replace />;
  }

  return children;
}
