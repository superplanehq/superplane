import { ArrowLeft, Plug } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { IntegrationSetupFlow } from "@/ui/integrations/IntegrationSetupFlow";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

interface IntegrationSetupProps {
  organizationId: string;
}

export function IntegrationSetup({ organizationId }: IntegrationSetupProps) {
  const navigate = useNavigate();
  const { integrationName } = useParams<{ integrationName: string }>();
  const { data: availableIntegrations = [] } = useAvailableIntegrations();

  const integrationDefinition = useMemo(() => {
    if (!integrationName) return undefined;
    return availableIntegrations.find((item) => item.name === integrationName);
  }, [availableIntegrations, integrationName]);
  const integrationLabel =
    getIntegrationTypeDisplayName(integrationDefinition?.label, integrationName) || integrationName || "Integration";
  const [currentName, setCurrentName] = useState(integrationLabel);
  const [currentStatus, setCurrentStatus] = useState<string | undefined>(undefined);
  const [isFinalStep, setIsFinalStep] = useState(false);

  if (!integrationName) {
    return (
      <div className="pt-6">
        <p className="text-gray-500 dark:text-gray-400">Integration not found</p>
      </div>
    );
  }

  return (
    <div className="pt-6 space-y-6">
      {!isFinalStep ? (
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate(`/${organizationId}/settings/integrations`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div className="flex items-center gap-3 flex-1 min-w-[200px]">
            <IntegrationIcon
              integrationName={integrationName}
              iconSlug={integrationDefinition?.icon}
              className="h-6 w-6 text-gray-500 dark:text-gray-400"
            />
            <h4 className="text-2xl font-medium">{currentName || integrationLabel}</h4>
          </div>
          {currentStatus ? (
            <div className="flex items-center gap-2 ml-auto">
              <Plug
                className={`w-4 h-4 ${
                  currentStatus === "ready"
                    ? "text-green-500"
                    : currentStatus === "error"
                      ? "text-red-600"
                      : "text-amber-600"
                }`}
              />
              <span
                className={`text-sm font-medium ${
                  currentStatus === "ready"
                    ? "text-green-500"
                    : currentStatus === "error"
                      ? "text-red-600"
                      : "text-amber-600"
                }`}
              >
                {currentStatus.charAt(0).toUpperCase() + currentStatus.slice(1)}
              </span>
            </div>
          ) : null}
        </div>
      ) : null}

      <IntegrationSetupFlow
        organizationId={organizationId}
        integrationName={integrationName}
        integrationDefinition={integrationDefinition}
        onCancel={() => navigate(`/${organizationId}/settings/integrations`)}
        onCompleted={(integrationId) => navigate(`/${organizationId}/settings/integrations/${integrationId}`)}
        onStateChange={(state) => {
          setCurrentName(state.name || integrationLabel);
          setCurrentStatus(state.status);
          setIsFinalStep(state.isFinalStep);
        }}
      />
    </div>
  );
}
