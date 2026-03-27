/**
 * Builds URLs to https://docs.superplane.com/components/…#anchor
 * matching scripts/generate_components_docs.go (integrationFilename + slugify).
 */

const DOCS_COMPONENTS_BASE = "https://docs.superplane.com/components";

const camelBoundary = /([a-z0-9])([A-Z])/g;

/** Fragment id for a component or trigger section (Go slugify). */
export function slugifyDocsAnchor(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "unknown";
  }
  let s = trimmed.replace(/_/g, "-");
  s = s.replace(camelBoundary, "$1-$2");
  s = s.replace(/ /g, "-");
  s = s.replace(/\./g, "-");
  return s.toLowerCase();
}

/**
 * Path segment for an integration docs page: label with spaces removed, lowercased
 * (e.g. "Google Cloud" → "googlecloud"), matching generated MDX filenames.
 */
function integrationDocsPathSegment(integrationLabel: string | undefined, integrationName: string | undefined): string {
  const label = integrationLabel?.trim() ?? "";
  if (label) {
    return label.replace(/\s+/g, "").toLowerCase();
  }
  if (integrationName) {
    return slugifyDocsAnchor(integrationName);
  }
  return "core";
}

export function buildComponentDocumentationUrl(params: {
  integrationName?: string;
  integrationLabel?: string;
  blockLabel: string;
}): string {
  const anchor = slugifyDocsAnchor(params.blockLabel);
  const segment = params.integrationName
    ? integrationDocsPathSegment(params.integrationLabel, params.integrationName)
    : "core";
  return `${DOCS_COMPONENTS_BASE}/${segment}#${anchor}`;
}

/** Context from the canvas node editor for docs URL resolution (keeps CanvasPage useMemo complexity low). */
export type SidebarComponentDocsEditingContext = {
  displayLabel?: string;
  integrationName?: string;
  integrationLabel?: string;
};

export function buildSidebarComponentDocsPayload(
  blockName: string,
  editingNodeData: SidebarComponentDocsEditingContext | null | undefined,
  parts: {
    label?: string;
    description?: string;
    examplePayload?: Record<string, unknown>;
    payloadLabel: "Example Output" | "Example Data";
  },
) {
  const blockLabel = parts.label || editingNodeData?.displayLabel || blockName;
  return {
    description: parts.description,
    examplePayload: parts.examplePayload,
    payloadLabel: parts.payloadLabel,
    documentationUrl: buildComponentDocumentationUrl({
      integrationName: editingNodeData?.integrationName,
      integrationLabel: editingNodeData?.integrationLabel,
      blockLabel,
    }),
  };
}
