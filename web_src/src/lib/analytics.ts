import { posthog } from "@/posthog";

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

  canvasDelete: (canvasId: string, organizationId: string, nodeCount: number) => {
    posthog.capture("canvas:canvas_delete", {
      canvas_id: canvasId,
      organization_id: organizationId,
      node_count: nodeCount,
    });
  },

  canvasRename: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:canvas_rename", { canvas_id: canvasId, organization_id: organizationId });
  },

  yamlExport: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:yaml_export", { canvas_id: canvasId, organization_id: organizationId });
  },

  yamlImport: () => {
    posthog.capture("canvas:yaml_import", {});
  },

  nodeAdd: (
    nodeType: string,
    integration: string | undefined,
    nodeRef: string | undefined,
    organizationId: string,
  ) => {
    posthog.capture("canvas:node_add", { node_type: nodeType, integration, node_ref: nodeRef, organization_id: organizationId });
  },

  nodeRemove: (
    nodeType: string,
    integration: string | undefined,
    nodeRef: string | undefined,
    organizationId: string,
  ) => {
    posthog.capture("canvas:node_remove", { node_type: nodeType, integration, node_ref: nodeRef, organization_id: organizationId });
  },

  nodeConfigure: (
    nodeType: string,
    integration: string | undefined,
    fieldCount: number,
    organizationId: string,
  ) => {
    posthog.capture("canvas:node_configure", { node_type: nodeType, integration, field_count: fieldCount, organization_id: organizationId });
  },

  versionPublish: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:version_publish", { canvas_id: canvasId, organization_id: organizationId });
  },

  integrationCreate: (integrationType: string, organizationId: string) => {
    posthog.capture("integration:integration_create", {
      integration_type: integrationType,
      organization_id: organizationId,
    });
  },

  orgCreate: (organizationId: string) => {
    posthog.capture("auth:org_create", { organization_id: organizationId });
  },
};
