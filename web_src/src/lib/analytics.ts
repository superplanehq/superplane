import { posthog } from "@/posthog";

export const analytics = {
  memberAccept: (organizationId: string) => {
    posthog.capture("settings:member_accept", { organization_id: organizationId });
  },

  canvasCreate: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:canvas_create", { canvas_id: canvasId, organization_id: organizationId });
  },

  canvasDelete: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas:canvas_delete", { canvas_id: canvasId, organization_id: organizationId });
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
