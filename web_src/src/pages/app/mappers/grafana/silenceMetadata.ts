import type { MetadataItem } from "@/ui/metadataList";

const FILTER_PREVIEW_MAX_LENGTH = 60;

export interface SilenceSelectionNodeMetadata {
  comment?: string;
  label?: string;
}

export function buildSilenceSelectionMetadata(
  nodeMetadata: SilenceSelectionNodeMetadata | undefined,
  silence: string | undefined,
): MetadataItem[] {
  const metadataLabel = nodeMetadata?.comment?.trim() || nodeMetadata?.label?.trim();
  if (metadataLabel) {
    return [{ icon: "bell-off", label: metadataLabel }];
  }

  const trimmedSilence = silence?.trim();
  if (!trimmedSilence) {
    return [];
  }

  return [{ icon: "bell-off", label: trimmedSilence }];
}

export function buildSilenceFilterMetadata(filter: string | undefined): MetadataItem[] {
  const trimmedFilter = filter?.trim();
  if (!trimmedFilter) {
    return [];
  }

  const preview =
    trimmedFilter.length > FILTER_PREVIEW_MAX_LENGTH
      ? trimmedFilter.slice(0, FILTER_PREVIEW_MAX_LENGTH).trimEnd() + "..."
      : trimmedFilter;

  return [{ icon: "filter", label: `Filter: ${preview}` }];
}
