import { useEffect, useRef } from "react";
import { posthog } from "@/posthog";
import type { OrganizationsIntegration } from "@/api-client";
// Tracks when a connect form was opened so duration_s can be computed on submit.
// Keyed by integration name; last-write-wins for the same integration.
const integrationConnectStartTimes = new Map<string, number>();

export const analytics = {
  memberAccept: (organizationId: string) => {
    posthog.capture("settings:member_accept", { organization_id: organizationId });
  },

  canvasCreate: (
    canvasId: string,
    organizationId: string,
    method: "ui" | "cli" | "yaml_import" | "template",
    templateId: string | undefined,
    hasDescription: boolean,
  ) => {
    posthog.capture("canvas:canvas_create", {
      canvas_id: canvasId,
      organization_id: organizationId,
      method,
      template_id: templateId,
      has_description: hasDescription,
    });
  },

  canvasView: (canvasId: string, nodeCount: number, edgeCount: number, organizationId: string) => {
    posthog.capture("canvas:canvas_view", {
      canvas_id: canvasId,
      node_count: nodeCount,
      edge_count: edgeCount,
      organization_id: organizationId,
    });
  },

  canvasDelete: (canvasId: string, organizationId: string, nodeCount: number) => {
    posthog.capture("canvas:canvas_delete", {
      canvas_id: canvasId,
      organization_id: organizationId,
      node_count: nodeCount,
    });
  },

  yamlExport: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:yaml_export", { canvas_id: canvasId, organization_id: organizationId });
  },

  yamlImport: () => {
    posthog.capture("canvas:yaml_import", {});
  },

  nodeAdd: (nodeType: string, integration: string | undefined, nodeRef: string | undefined, organizationId: string) => {
    posthog.capture("canvas:node_add", {
      node_type: nodeType,
      integration,
      node_ref: nodeRef,
      organization_id: organizationId,
    });
  },

  nodeRemove: (
    nodeType: string,
    integration: string | undefined,
    nodeRef: string | undefined,
    organizationId: string,
  ) => {
    posthog.capture("canvas:node_remove", {
      node_type: nodeType,
      integration,
      node_ref: nodeRef,
      organization_id: organizationId,
    });
  },

  nodeConfigure: (nodeType: string, integration: string | undefined, fieldCount: number, organizationId: string) => {
    posthog.capture("canvas:node_configure", {
      node_type: nodeType,
      integration,
      field_count: fieldCount,
      organization_id: organizationId,
    });
  },

  edgeCreate: (organizationId: string) => {
    posthog.capture("canvas:edge_create", { organization_id: organizationId });
  },

  edgeRemove: (organizationId: string) => {
    posthog.capture("canvas:edge_remove", { organization_id: organizationId });
  },

  autoLayout: (nodeCount: number, organizationId: string) => {
    posthog.capture("canvas:auto_layout", { node_count: nodeCount, organization_id: organizationId });
  },

  eventEmit: (nodeType: string, integration: string | undefined, organizationId: string) => {
    posthog.capture("canvas:event_emit", { node_type: nodeType, integration, organization_id: organizationId });
  },

  versionPublish: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:version_publish", { canvas_id: canvasId, organization_id: organizationId });
  },

  integrationConnectStart: (
    integration: string,
    source: "node_configuration" | "integrations_page",
    organizationId: string,
  ) => {
    integrationConnectStartTimes.set(integration, Date.now());
    posthog.capture("integration:connect_start", {
      integration,
      source,
      organization_id: organizationId,
    });
  },

  orgCreate: (organizationId: string) => {
    posthog.capture("auth:org_create", { organization_id: organizationId });
  },

  canvasRunItemOpen: (nodeRef: string | undefined, executionStatus: string, organizationId: string) => {
    posthog.capture("canvas:run_item_open", {
      node_ref: nodeRef,
      execution_status: executionStatus,
      organization_id: organizationId,
    });
  },

  canvasRunItemTabView: (tab: "details" | "payload" | "config", organizationId: string) => {
    posthog.capture("canvas:run_item_tab_view", { tab, organization_id: organizationId });
  },

  canvasComponentError: (nodeRef: string | undefined, errorMessage: string, organizationId: string) => {
    posthog.capture("canvas:component_error", {
      node_ref: nodeRef,
      error_message: errorMessage,
      organization_id: organizationId,
    });
  },

  canvasLogView: (organizationId: string) => {
    posthog.capture("canvas:log_view", { organization_id: organizationId });
  },

  integrationConnectSubmit: (
    integration: string,
    source: "node_configuration" | "integrations_page",
    status: "ready" | "error" | "pending",
    organizationId: string,
  ) => {
    const startTime = integrationConnectStartTimes.get(integration);
    const durationS = startTime !== undefined ? (Date.now() - startTime) / 1000 : undefined;
    integrationConnectStartTimes.delete(integration);
    posthog.capture("integration:connect_submit", {
      integration,
      source,
      status,
      duration_s: durationS,
      organization_id: organizationId,
    });
  },

  integrationConfigureOpen: (
    integration: string,
    source: "node_configuration" | "integrations_page",
    previousStatus: "ready" | "error" | "pending",
    organizationId: string,
  ) => {
    posthog.capture("integration:configure_open", {
      integration,
      source,
      previous_status: previousStatus,
      organization_id: organizationId,
    });
  },

  integrationDelete: (integration: string, organizationId: string) => {
    posthog.capture("integration:integration_delete", {
      integration,
      organization_id: organizationId,
    });
  },
};

export function useIntegrationConfigureOpen(
  integration: OrganizationsIntegration | undefined,
  integrationId: string | null | undefined,
  source: "node_configuration" | "integrations_page",
  organizationId: string | undefined,
) {
  const firedRef = useRef<string | null>(null);

  useEffect(() => {
    if (!integration || !integrationId || !organizationId) return;
    if (firedRef.current === integrationId) return;
    const integrationTypeName = integration.spec?.integrationName;
    if (!integrationTypeName) return;
    const previousStatus = (integration.status?.state || "pending") as "ready" | "error" | "pending";
    analytics.integrationConfigureOpen(integrationTypeName, source, previousStatus, organizationId);
    firedRef.current = integrationId;
  }, [integration, integrationId, source, organizationId]);

  useEffect(() => {
    if (!integrationId) firedRef.current = null;
  }, [integrationId]);
}
