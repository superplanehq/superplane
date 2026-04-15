import { posthog } from "@/posthog";

export const analytics = {
  organizationCreated: (organizationId: string) => {
    posthog.capture("organization created", { organization_id: organizationId });
  },

  canvasCreated: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas created", { canvas_id: canvasId, organization_id: organizationId });
  },

  canvasDeleted: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas deleted", { canvas_id: canvasId, organization_id: organizationId });
  },

  integrationConnected: (integrationType: string, organizationId: string) => {
    posthog.capture("integration connected", { integration_type: integrationType, organization_id: organizationId });
  },

  organizationJoined: (organizationId: string) => {
    posthog.capture("organization joined", { organization_id: organizationId });
  },

  canvasPublished: (canvasId: string, organizationId: string) => {
    posthog.capture("canvas published", { canvas_id: canvasId, organization_id: organizationId });
  },
};
