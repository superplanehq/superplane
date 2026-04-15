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
};
