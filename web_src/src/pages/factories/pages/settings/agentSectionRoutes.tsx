import { Outlet, Route } from "react-router";

import { RequirePermission } from "@/components/PermissionGate";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactorySettingsAgentPage } from "./FactorySettingsAgentPage";
import { FactorySettingsMCPConnectionPage } from "./FactorySettingsMCPConnectionPage";
import { FactorySettingsSkillEditorPage } from "./FactorySettingsSkillEditorPage";

/**
 * Agent and its pages (MCP connection, skill editor) share the workspace
 * shell. They stay off the settings layout so the settings sidebar does not
 * open beside them.
 */
export const agentSectionRoute = (
  <Route
    path="agent"
    element={
      <RequirePermission resource="factories" action="update">
        <RequireExperimentalFeature anyOf={[FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS]}>
          <div className="min-h-full bg-sidebar dark:bg-background">
            <Outlet />
          </div>
        </RequireExperimentalFeature>
      </RequirePermission>
    }
  >
    <Route index element={<FactorySettingsAgentPage />} />
    <Route
      path="mcp/:resourceId"
      element={
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_MCP}>
          <FactorySettingsMCPConnectionPage />
        </RequireExperimentalFeature>
      }
    />
    <Route
      path="skills/new"
      element={
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_SKILLS}>
          <FactorySettingsSkillEditorPage />
        </RequireExperimentalFeature>
      }
    />
    <Route
      path="skills/:resourceId"
      element={
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_SKILLS}>
          <FactorySettingsSkillEditorPage />
        </RequireExperimentalFeature>
      }
    />
  </Route>
);
