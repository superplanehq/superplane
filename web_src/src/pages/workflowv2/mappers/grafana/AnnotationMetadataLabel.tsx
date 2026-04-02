import { useIntegrationResources } from "@/hooks/useIntegrations";

/** Must match pkg/integrations/grafana resourceTypeAnnotation. */
export const GRAFANA_ANNOTATION_RESOURCE_TYPE = "annotation";

type AnnotationMetadataLabelProps = {
  organizationId?: string;
  integrationId?: string;
  annotationId: string;
};

/**
 * Resolves a stored annotation ID to the Grafana picker label (e.g. "#42 · text") for canvas metadata.
 */
export function AnnotationMetadataLabel({ organizationId, integrationId, annotationId }: AnnotationMetadataLabelProps) {
  const { data: resources } = useIntegrationResources(
    organizationId ?? "",
    integrationId ?? "",
    GRAFANA_ANNOTATION_RESOURCE_TYPE,
  );

  const name = resources?.find((r) => r.id === annotationId)?.name?.trim();
  const display = name || annotationId;

  return <span className="truncate">Annotation: {display}</span>;
}
