import type { ComponentsNode } from "@/api-client";
import { INTEGRATION_APP_LOGO_MAP } from "@/ui/componentSidebar/integrationIcons";

const TEMPLATE_TAG_MAP: Record<string, string[]> = {
  "Incident Router": ["Incident Management", "AI"],
  "Incident Data Collection": ["Incident Management", "Observability"],
  "Health Check Monitor": ["Monitoring"],
  "Staged Release": ["CI/CD", "Deployment"],
  "Multi-repo CI and release": ["CI/CD", "Deployment"],
  "Automated Rollback": ["CI/CD", "Deployment", "Observability"],
  "Policy Gated Deployment": ["CI/CD", "Deployment", "Security"],
};

export const ALL_TAGS: string[] = [...new Set(Object.values(TEMPLATE_TAG_MAP).flat())].sort();

export function getTemplateTags(templateName: string | undefined): string[] {
  if (!templateName) return [];
  return TEMPLATE_TAG_MAP[templateName] ?? [];
}

/**
 * Extracts unique integration identifiers from a list of nodes by parsing
 * the block type name (e.g. "github.runWorkflow" -> "github").
 * Only returns integrations that have a known icon in INTEGRATION_APP_LOGO_MAP.
 */
export function extractIntegrations(nodes: ComponentsNode[] | undefined): string[] {
  if (!nodes) return [];

  const integrations = new Set<string>();

  for (const node of nodes) {
    const blockName = node.component?.name || node.trigger?.name;
    if (!blockName) continue;

    const rawPrefix = blockName.split(".")[0];
    if (!rawPrefix || rawPrefix === blockName) continue;

    const prefix = rawPrefix.toLowerCase();
    if (INTEGRATION_APP_LOGO_MAP[prefix]) {
      integrations.add(prefix);
    }
  }

  return [...integrations].sort();
}

export function countNodesByType(nodes: ComponentsNode[] | undefined): {
  components: number;
  triggers: number;
} {
  if (!nodes) return { components: 0, triggers: 0 };

  let components = 0;
  let triggers = 0;

  for (const node of nodes) {
    if (node.type === "TYPE_COMPONENT") components++;
    else if (node.type === "TYPE_TRIGGER") triggers++;
  }

  return { components, triggers };
}
