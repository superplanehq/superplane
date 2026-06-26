export type IntegrationSource = "node_configuration" | "integrations_page" | "install_wizard";

import { useEffect, useRef } from "react";
import { posthog } from "@/posthog";
import type { OrganizationsIntegration } from "@/api-client";
import { getUtmEventProperties } from "@/lib/utmAttribution";
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

  integrationRequested: (organizationId: string) => {
    posthog.capture("request_integration_clicked", { organization_id: organizationId });
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

  agentMessageSendSubmitted: (
    chatId: string,
    canvasId: string | undefined,
    organizationId: string | undefined,
    mode: string | undefined,
  ) => {
    posthog.capture("agent:message_send_submitted", {
      chat_id: chatId,
      canvas_id: canvasId,
      organization_id: organizationId,
      mode,
    });
  },

  agentMessageSendAcknowledged: (
    chatId: string,
    canvasId: string | undefined,
    organizationId: string | undefined,
    mode: string | undefined,
    durationMs: number,
  ) => {
    posthog.capture("agent:message_send_acknowledged", {
      chat_id: chatId,
      canvas_id: canvasId,
      organization_id: organizationId,
      mode,
      duration_ms: durationMs,
    });
  },

  agentMessageSendFailed: (
    chatId: string,
    canvasId: string | undefined,
    organizationId: string | undefined,
    mode: string | undefined,
    durationMs: number,
  ) => {
    posthog.capture("agent:message_send_failed", {
      chat_id: chatId,
      canvas_id: canvasId,
      organization_id: organizationId,
      mode,
      duration_ms: durationMs,
    });
  },

  integrationConnectStart: (integration: string, source: IntegrationSource, organizationId: string) => {
    integrationConnectStartTimes.set(integration, Date.now());
    posthog.capture("integration:connect_start", {
      integration,
      source,
      organization_id: organizationId,
    });
  },

  orgCreate: (organizationId: string) => {
    posthog.capture("auth:org_create", {
      organization_id: organizationId,
      ...getUtmEventProperties(),
    });
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
    source: IntegrationSource,
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
    source: IntegrationSource,
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

  surveySent: (surveyId: string, surveyName: string, responseProps: Record<string, string | string[]>) => {
    posthog.capture("survey sent", {
      $survey_id: surveyId,
      $survey_name: surveyName,
      ...responseProps,
      $survey_completed: true,
    });
  },

  surveyDismissed: (surveyId: string) => {
    posthog.capture("survey dismissed", { $survey_id: surveyId });
  },
};

export function useIntegrationConfigureOpen(
  integration: OrganizationsIntegration | undefined,
  integrationId: string | null | undefined,
  source: IntegrationSource,
  organizationId: string | undefined,
) {
  const firedRef = useRef<string | null>(null);

  useEffect(() => {
    if (!integration || !integrationId || !organizationId) return;
    if (firedRef.current === integrationId) return;
    const integrationTypeName = integration.metadata?.integrationName;
    if (!integrationTypeName) return;
    const previousStatus = (integration.status?.state || "pending") as "ready" | "error" | "pending";
    analytics.integrationConfigureOpen(integrationTypeName, source, previousStatus, organizationId);
    firedRef.current = integrationId;
  }, [integration, integrationId, source, organizationId]);

  useEffect(() => {
    if (!integrationId) firedRef.current = null;
  }, [integrationId]);
}
