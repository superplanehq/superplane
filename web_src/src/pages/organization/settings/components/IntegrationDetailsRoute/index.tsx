import { useIntegration } from "@/hooks/useIntegrations";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { ArrowLeft, Loader2 } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { LegacyIntegrationDetails } from "./LegacyIntegrationDetails";
import { CapabilityBasedIntegrationDetails } from "../CapabilityBasedIntegrationDetails";
import { isCapabilityBasedIntegration } from "@/lib/integrations";

interface IntegrationDetailsRouteProps {
  organizationId: string;
}

export function IntegrationDetailsRoute({ organizationId }: IntegrationDetailsRouteProps) {
  const { integrationId } = useParams<{ integrationId: string }>();
  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");
  const integrationsHref = `/${organizationId}/settings/integrations`;

  useReportPageReady(!isLoading, {
    failed: !!(error || !integration),
  });

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <Link
            to={integrationsHref}
            className="text-content-secondary hover:text-content-primary"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-content-secondary" />
        </div>
      </div>
    );
  }

  if (error || !integration) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <Link
            to={integrationsHref}
            className="text-content-secondary hover:text-content-primary"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="bg-surface-raised rounded-lg border border-edge-default p-6">
          <p className="text-content-secondary">Integration not found</p>
        </div>
      </div>
    );
  }

  if (isCapabilityBasedIntegration(integration)) {
    return <CapabilityBasedIntegrationDetails organizationId={organizationId} integration={integration} />;
  }

  return <LegacyIntegrationDetails organizationId={organizationId} integration={integration} />;
}
