import githubIcon from "@/assets/icons/integrations/github.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import type { IntegrationsIntegration } from "../../../api-client/types.gen";
import { Icon } from "../../../components/Icon";
import { IntegrationModal } from "../../../components/IntegrationZeroState/IntegrationModal";
import { useIntegrations } from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  const navigate = useNavigate();
  const [selectedIntegrationType, setSelectedIntegrationType] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isIntegrationSelectorOpen, setIsIntegrationSelectorOpen] = useState(false);
  const [editingIntegration, setEditingIntegration] = useState<IntegrationsIntegration | null>(null);

  const { data: integrations = [], isLoading: integrationsLoading } = useIntegrations(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION",
  );

  const integrationTypes = [
    { id: "github", name: "GitHub", icon: githubIcon },
    { id: "semaphore", name: "Semaphore", icon: SemaphoreLogo },
  ];

  const handleCreateIntegration = (type: string) => {
    setSelectedIntegrationType(type);
    setIsIntegrationSelectorOpen(false);
    setIsModalOpen(true);
  };

  const handleAddIntegrationClick = () => {
    setIsIntegrationSelectorOpen(true);
  };

  const handleModalClose = () => {
    setIsModalOpen(false);
    setSelectedIntegrationType(null);
    setEditingIntegration(null);
  };

  const handleIntegrationSuccess = () => {
    setIsModalOpen(false);
    setSelectedIntegrationType(null);
    setEditingIntegration(null);
  };

  const handleEditIntegration = (integration: IntegrationsIntegration) => {
    setEditingIntegration(integration);
    setSelectedIntegrationType(integration.spec?.type || "");
    setIsModalOpen(true);
  };

  if (integrationsLoading) {
    return (
      <div className="pt-6">
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">Loading integrations...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      {/* Deprecation Warning */}
      <div className="mb-6 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <Icon name="triangle-alert" size="sm" className="text-amber-600 dark:text-amber-400 mt-0.5 flex-shrink-0" />
          <div className="flex-1">
            <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-200 mb-1">
              Integrations are deprecated
            </h3>
            <p className="text-sm text-amber-800 dark:text-amber-300 mb-3">
              The integrations system has been replaced with Applications, which provide enhanced functionality and
              better integration with external systems. Please migrate to Applications for new connections.
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate(`/${organizationId}/settings/applications`)}
              className="text-amber-900 dark:text-amber-200 border-amber-300 dark:border-amber-700 hover:bg-amber-100 dark:hover:bg-amber-900/30"
            >
              Go to Applications
            </Button>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
        <div className="p-6">
          {integrations.length === 0 ? (
            <div className="text-center py-12">
              <Icon name="integration_instructions" size="lg" className="text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-800 dark:text-gray-100 mb-2">No integrations yet</h3>
              <p className="text-gray-500 dark:text-gray-400 mb-6">
                Connect external services to streamline your workflow
              </p>
              <Button onClick={handleAddIntegrationClick} className="flex items-center gap-2">
                <Icon name="plus" size="sm" />
                Add Integration
              </Button>
            </div>
          ) : (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-medium">Organization Integrations</h2>
                <Button onClick={handleAddIntegrationClick} className="flex items-center gap-2">
                  <Icon name="plus" size="sm" />
                  Add Integration
                </Button>
              </div>

              <div className="space-y-4">
                {integrations.map((integration) => (
                  <div
                    key={integration.metadata?.id}
                    className="border border-gray-300 dark:border-gray-700 rounded-lg p-4"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex-shrink-0">
                          <img
                            src={integration.spec?.type === "github" ? githubIcon : SemaphoreLogo}
                            alt={integration.spec?.type}
                            className="w-5 "
                          />
                        </div>
                        <div>
                          <h3 className="font-medium text-gray-800 dark:text-gray-100">{integration.metadata?.name}</h3>
                          <p className="text-sm text-gray-500 dark:text-gray-400">{integration.spec?.url}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="px-2 py-1 text-xs font-medium bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded-full">
                          {integration.spec?.type}
                        </span>
                        <button
                          data-testid={`edit-integration-${integration.metadata?.name || ""}`}
                          onClick={() => handleEditIntegration(integration)}
                          className="p-1 text-gray-500 hover:text-gray-800 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"
                          title="Edit integration"
                        >
                          <Icon name="edit" size="sm" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Integration Type Selector */}
      {isIntegrationSelectorOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-medium text-gray-800 dark:text-gray-100">Select Integration Type</h3>
                <button
                  onClick={() => setIsIntegrationSelectorOpen(false)}
                  className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                >
                  <Icon name="close" size="sm" />
                </button>
              </div>
              <div className="space-y-3">
                {integrationTypes.map((type) => (
                  <button
                    key={type.id}
                    onClick={() => handleCreateIntegration(type.id)}
                    className="w-full flex items-center gap-3 p-4 text-left border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    <div className="flex-shrink-0">
                      <img src={type.icon} alt={type.name} className="w-5" />
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-800 dark:text-gray-100">{type.name}</h4>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {type.id === "github" ? "Connect to GitHub repositories" : "Connect to Semaphore CI/CD"}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {selectedIntegrationType && (
        <IntegrationModal
          open={isModalOpen}
          onClose={handleModalClose}
          integrationType={selectedIntegrationType}
          canvasId={""}
          organizationId={organizationId}
          onSuccess={handleIntegrationSuccess}
          domainType="DOMAIN_TYPE_ORGANIZATION"
          editingIntegration={editingIntegration || undefined}
        />
      )}
    </div>
  );
}
