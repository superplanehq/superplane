import { useIntegration } from "@/hooks/useIntegrations";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { consumeIntegrationSetupReturn, peekIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useEffect } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { LegacyIntegrationDetails } from "./LegacyIntegrationDetails";
import { CapabilityBasedIntegrationDetails } from "../CapabilityBasedIntegrationDetails";
import { isCapabilityBasedIntegration } from "@/lib/integrations";

interface IntegrationDetailsRouteProps {
  organizationId: string;
}

export function IntegrationDetailsRoute({ organizationId }: IntegrationDetailsRouteProps) {
  const { integrationId } = useParams<{ integrationId: string }>();
  const navigate = useNavigate();
  const returnTo = peekIntegrationSetupReturn(organizationId, integrationId || "");
  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");
  const integrationsHref = `/${organizationId}/settings/integrations`;

  useEffect(() => {
    if (!integrationId || !returnTo) return;
    consumeIntegrationSetupReturn(organizationId, integrationId);
    navigate(returnTo, { replace: true });
  }, [integrationId, navigate, organizationId, returnTo]);

  useReportPageReady(!isLoading, {
    failed: !!(error || !integration),
  });

  if (isLoading || returnTo) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <Link
            to={integrationsHref}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-gray-500 dark:text-gray-400" />
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
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <p className="text-gray-500 dark:text-gray-400">Integration not found</p>
        </div>
      </div>
    );
  }

  if (isCapabilityBasedIntegration(integration)) {
    return <CapabilityBasedIntegrationDetails organizationId={organizationId} integration={integration} />;
  }

  return <LegacyIntegrationDetails organizationId={organizationId} integration={integration} />;
}
