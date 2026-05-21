import type { ExperimentalFeature } from "@/hooks/useExperimentalFeatures";

/**
 * Frontend overrides for the user-facing label/description of experimental
 * features whose product name differs from the backend registry id. Useful
 * when a feature has been rebranded without renaming the stored feature id.
 */
const FEATURE_DISPLAY_OVERRIDES: Record<string, { label: string; description: string }> = {
  // Backend id is still "dashboards"; the product surfaces this as "Console".
  dashboards: {
    label: "Console",
    description: "Console panels and widgets on canvases",
  },
};

export function getExperimentalFeatureLabel(feature: Pick<ExperimentalFeature, "id" | "label">): string {
  return FEATURE_DISPLAY_OVERRIDES[feature.id]?.label ?? feature.label;
}

export function getExperimentalFeatureDescription(feature: Pick<ExperimentalFeature, "id" | "description">): string {
  return FEATURE_DISPLAY_OVERRIDES[feature.id]?.description ?? feature.description;
}
