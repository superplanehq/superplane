import { useState } from "react";
import { Button } from "../../../components/Button/button";
import { MaterialSymbol } from "../../../components/MaterialSymbol/material-symbol";
import { useIntegrations } from "../../../hooks/useIntegrations";
import { IntegrationModal } from "../../../components/IntegrationZeroState/IntegrationModal";
import type { IntegrationsIntegration } from "../../../api-client/types.gen";

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  const [selectedIntegrationType, setSelectedIntegrationType] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isIntegrationSelectorOpen, setIsIntegrationSelectorOpen] = useState(false);
  const [editingIntegration, setEditingIntegration] = useState<IntegrationsIntegration | null>(null);

  const { data: integrations = [], isLoading: integrationsLoading } = useIntegrations(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION",
  );

  const integrationTypes = [
    { id: "github", name: "GitHub", icon: "code" },
    { id: "semaphore", name: "Semaphore", icon: "build" },
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
        <h1 className="text-2xl font-semibold mb-6">Integrations</h1>
        <div className="flex justify-center items-center h-32">
          <p className="text-zinc-500 dark:text-zinc-400">Loading integrations...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="flex justify-between items-start mb-6">
        <div>
          <h1 className="text-2xl font-semibold">Integrations</h1>
          <p className="text-zinc-600 dark:text-zinc-400 mt-2">Manage integrations for your organization</p>
        </div>
      </div>

      <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
        <div className="p-6">
          {integrations.length === 0 ? (
            <div className="text-center py-12">
              <MaterialSymbol name="integration_instructions" size="lg" className="text-zinc-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-2">No integrations yet</h3>
              <p className="text-zinc-600 dark:text-zinc-400 mb-6">
                Connect external services to streamline your workflow
              </p>
              <Button color="blue" onClick={handleAddIntegrationClick} className="flex items-center gap-2">
                <MaterialSymbol name="add" size="sm" />
                Add Integration
              </Button>
            </div>
          ) : (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-medium">Organization Integrations</h2>
                <Button color="blue" onClick={handleAddIntegrationClick} className="flex items-center gap-2">
                  <MaterialSymbol name="add" size="sm" />
                  Add Integration
                </Button>
              </div>

              <div className="space-y-4">
                {integrations.map((integration) => (
                  <div
                    key={integration.metadata?.id}
                    className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <MaterialSymbol
                          name={integration.spec?.type === "github" ? "code" : "build"}
                          className="text-zinc-600 dark:text-zinc-400"
                        />
                        <div>
                          <h3 className="font-medium text-zinc-900 dark:text-zinc-100">{integration.metadata?.name}</h3>
                          <p className="text-sm text-zinc-600 dark:text-zinc-400">{integration.spec?.url}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="px-2 py-1 text-xs font-medium bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded-full">
                          {integration.spec?.type}
                        </span>
                        <button
                          onClick={() => handleEditIntegration(integration)}
                          className="p-1 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded"
                          title="Edit integration"
                        >
                          <MaterialSymbol name="edit" size="sm" />
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
          <div className="bg-white dark:bg-zinc-900 rounded-lg shadow-xl border border-zinc-200 dark:border-zinc-800 max-w-md w-full mx-4">
            <div className="p-6">
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100">Select Integration Type</h3>
                <button
                  onClick={() => setIsIntegrationSelectorOpen(false)}
                  className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                >
                  <MaterialSymbol name="close" size="sm" />
                </button>
              </div>
              <div className="space-y-3">
                {integrationTypes.map((type) => (
                  <button
                    key={type.id}
                    onClick={() => handleCreateIntegration(type.id)}
                    className="w-full flex items-center gap-3 p-4 text-left border border-zinc-200 dark:border-zinc-700 rounded-lg hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
                  >
                    <MaterialSymbol name={type.icon} className="text-zinc-600 dark:text-zinc-400" />
                    <div>
                      <h4 className="font-medium text-zinc-900 dark:text-zinc-100">{type.name}</h4>
                      <p className="text-sm text-zinc-600 dark:text-zinc-400">
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
