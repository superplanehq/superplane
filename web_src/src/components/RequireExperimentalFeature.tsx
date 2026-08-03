import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import type { ReactNode } from "react";
import { Navigate, useParams } from "react-router-dom";

interface RequireExperimentalFeatureProps {
  featureId: string;
  children: ReactNode;
}

export function RequireExperimentalFeature({ featureId, children }: RequireExperimentalFeatureProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { has } = useExperimentalFeature(organizationId);

  if (!has(featureId)) {
    return <Navigate to={organizationId ? `/${organizationId}` : "/"} replace />;
  }

  return children;
}
