import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useWorkspaceLoading } from "@/hooks/useWorkspaceLoading";
import { WORKSPACE_LOADING_COPY } from "@/lib/workspaceLoadingCopy";
import type { ReactNode } from "react";
import { Navigate, useParams } from "react-router";

interface RequireExperimentalFeatureProps {
  featureId?: string;
  anyOf?: string[];
  children: ReactNode;
}

export function RequireExperimentalFeature({ featureId, anyOf, children }: RequireExperimentalFeatureProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { has, isLoading } = useExperimentalFeature(organizationId);
  const overlayHandles = useWorkspaceLoading(WORKSPACE_LOADING_COPY.workspace, isLoading);
  const requiredIds = anyOf?.length ? anyOf : featureId ? [featureId] : [];

  if (isLoading) {
    if (overlayHandles) {
      return null;
    }
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  if (!requiredIds.some((id) => has(id))) {
    return <Navigate to={organizationId ? `/${organizationId}` : "/"} replace />;
  }

  return children;
}
