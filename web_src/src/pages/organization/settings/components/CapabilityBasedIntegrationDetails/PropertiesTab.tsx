import type { IntegrationProperty } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import type { Dispatch, SetStateAction } from "react";

export interface PropertiesTabProps {
  integrationProperties: IntegrationProperty[];
  propertyDrafts: Record<string, string>;
  setPropertyDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  settingsMutationBusy: boolean;
  saveProperty: (propertyName: string, value: string) => Promise<void>;
  isSavingProperty: (propertyName: string | undefined) => boolean;
}

export function PropertiesTab({
  integrationProperties,
  propertyDrafts,
  setPropertyDrafts,
  canUpdateIntegrations,
  permissionsLoading,
  settingsMutationBusy,
  saveProperty,
  isSavingProperty,
}: PropertiesTabProps) {
  if (integrationProperties.length === 0) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No properties for this integration.</p>;
  }

  return (
    <div className="space-y-6 rounded-lg border border-gray-300 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
      {integrationProperties.map((property) => {
        const propertyName = property.name!;
        const title = property.label!;
        const isEditable = property.editable === true;
        const draft = propertyDrafts[propertyName] ?? "";
        const currentValue = property.value ?? "";
        const propertyDirty = draft !== currentValue;
        const savingThisProperty = isSavingProperty(propertyName);

        return (
          <div
            key={propertyName}
            className="border-b border-gray-200 pb-6 last:border-b-0 last:pb-0 dark:border-gray-800"
          >
            <div className="text-sm font-medium text-gray-800 dark:text-gray-100">{title}</div>
            {property.description ? (
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{property.description}</p>
            ) : null}
            <div className="mt-3">
              {isEditable ? (
                <PermissionTooltip
                  allowed={canUpdateIntegrations || permissionsLoading}
                  message="You don't have permission to update integrations."
                >
                  <div className="flex max-w-xl flex-col gap-2 sm:max-w-none sm:flex-row sm:items-center sm:gap-3">
                    <Input
                      value={draft}
                      onChange={(event) =>
                        setPropertyDrafts((previous) => ({
                          ...previous,
                          [propertyName]: event.target.value,
                        }))
                      }
                      disabled={!canUpdateIntegrations || settingsMutationBusy}
                      className="w-full sm:max-w-xl sm:flex-1"
                    />
                    {propertyName ? (
                      <LoadingButton
                        type="button"
                        color="blue"
                        size="sm"
                        className="shrink-0"
                        disabled={!canUpdateIntegrations || !propertyDirty || settingsMutationBusy}
                        loading={savingThisProperty}
                        loadingText="Saving…"
                        onClick={() => void saveProperty(propertyName, draft)}
                      >
                        Save
                      </LoadingButton>
                    ) : null}
                  </div>
                </PermissionTooltip>
              ) : (
                <p className="text-sm text-gray-800 dark:text-gray-100">
                  {draft.trim() ? draft : <span className="text-gray-400 dark:text-gray-500">No value</span>}
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
