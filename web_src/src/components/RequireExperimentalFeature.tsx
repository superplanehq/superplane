import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import type { ReactNode } from "react";
import { Navigate, useParams } from "react-router";

interface RequireExperimentalFeatureProps {
  featureId: string;
  children: ReactNode;
}

export function RequireExperimentalFeature({ featureId, children }: RequireExperimentalFeatureProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { has, isLoading } = useExperimentalFeature(organizationId);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  if (!has(featureId)) {
    return <Navigate to={organizationId ? `/${organizationId}` : "/"} replace />;
  }

  return children;
}
