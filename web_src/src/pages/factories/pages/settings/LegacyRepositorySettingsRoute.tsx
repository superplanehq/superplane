import { Navigate, useParams } from "react-router";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useWorkspaceLoading } from "@/hooks/useWorkspaceLoading";
import { FEATURE_FACTORY_MULTI_REPO_TASKS } from "@/lib/experimentalFeatures";
import { WORKSPACE_LOADING_COPY } from "@/lib/workspaceLoadingCopy";
import { factoryAgentPath } from "../../lib/factoryPagePaths";
import { FactorySettingsRepositoryPage } from "./FactorySettingsRepositoryPage";

export function LegacyRepositorySettingsRoute() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  const { has, isLoading } = useExperimentalFeature(organizationId);
  const overlayHandles = useWorkspaceLoading(WORKSPACE_LOADING_COPY.workspace, isLoading);

  if (isLoading) {
    if (overlayHandles) return null;
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  if (has(FEATURE_FACTORY_MULTI_REPO_TASKS)) {
    return <Navigate to={factoryAgentPath(organizationId, factoryKey)} replace />;
  }

  return <FactorySettingsRepositoryPage />;
}
