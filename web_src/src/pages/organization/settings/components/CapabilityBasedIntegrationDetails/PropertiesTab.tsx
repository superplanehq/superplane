import type { IntegrationProperty } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { isUrl } from "@/lib/utils";
import { Check, Info, Pencil, X } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { useState } from "react";

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

type PropertyDescriptionTooltipProps = {
  title: string;
  description: string | undefined;
};

function PropertyDescriptionTooltip({ title, description }: PropertyDescriptionTooltipProps) {
  if (!description) {
    return null;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          aria-label={`About ${title}`}
        >
          <Info className="size-4 shrink-0" aria-hidden />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-xs text-balance">
        {description}
      </TooltipContent>
    </Tooltip>
  );
}

type PropertyReadonlyDisplayProps = {
  draft: string;
  trimmedDraft: string;
  readonlyHref: string | null;
};

function PropertyReadonlyDisplay({ draft, trimmedDraft, readonlyHref }: PropertyReadonlyDisplayProps) {
  return (
    <div className="min-w-0 max-w-xl text-sm">
      {readonlyHref ? (
        <a
          href={readonlyHref}
          target="_blank"
          rel="noopener noreferrer"
          className="break-all text-primary [text-decoration:underline!important] [text-underline-offset:2px]"
        >
          {readonlyHref}
        </a>
      ) : trimmedDraft === "" ? (
        <span className="text-gray-400 dark:text-gray-500">No value</span>
      ) : (
        <span className="text-gray-800 dark:text-gray-100">{draft}</span>
      )}
    </div>
  );
}

type PropertyEditingControlsProps = {
  inputId: string;
  propertyName: string;
  draft: string;
  propertyDirty: boolean;
  savingThisProperty: boolean;
  canUpdateIntegrations: boolean;
  settingsMutationBusy: boolean;
  setPropertyDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  setEditingPropertyName: Dispatch<SetStateAction<string | null>>;
  exitEdit: () => void;
  saveProperty: (propertyName: string, value: string) => Promise<void>;
};

function PropertyEditingControls({
  inputId,
  propertyName,
  draft,
  propertyDirty,
  savingThisProperty,
  canUpdateIntegrations,
  settingsMutationBusy,
  setPropertyDrafts,
  setEditingPropertyName,
  exitEdit,
  saveProperty,
}: PropertyEditingControlsProps) {
  return (
    <>
      <Input
        id={inputId}
        value={draft}
        onChange={(event) =>
          setPropertyDrafts((previous) => ({
            ...previous,
            [propertyName]: event.target.value,
          }))
        }
        disabled={!canUpdateIntegrations || settingsMutationBusy}
        className="min-w-0 flex-1 sm:max-w-xl"
      />
      <div className="flex shrink-0 items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Cancel editing"
          disabled={!canUpdateIntegrations || settingsMutationBusy || savingThisProperty}
          onClick={exitEdit}
        >
          <X className="size-4" aria-hidden />
        </Button>
        <LoadingButton
          type="button"
          color="blue"
          size="sm"
          className="shrink-0"
          disabled={!canUpdateIntegrations || !propertyDirty || settingsMutationBusy}
          loading={savingThisProperty}
          loadingText="Updating…"
          onClick={async () => {
            try {
              await saveProperty(propertyName, draft);
              setEditingPropertyName((current) => (current === propertyName ? null : current));
            } catch {
              // Toast already shown by saveProperty
            }
          }}
        >
          Update
          <Check className="size-4" aria-hidden />
        </LoadingButton>
      </div>
    </>
  );
}

type IntegrationPropertyRowProps = {
  property: IntegrationProperty;
  propertyDrafts: Record<string, string>;
  setPropertyDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  editingPropertyName: string | null;
  setEditingPropertyName: Dispatch<SetStateAction<string | null>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  settingsMutationBusy: boolean;
  saveProperty: (propertyName: string, value: string) => Promise<void>;
  isSavingProperty: (propertyName: string | undefined) => boolean;
};

function IntegrationPropertyRow({
  property,
  propertyDrafts,
  setPropertyDrafts,
  editingPropertyName,
  setEditingPropertyName,
  canUpdateIntegrations,
  permissionsLoading,
  settingsMutationBusy,
  saveProperty,
  isSavingProperty,
}: IntegrationPropertyRowProps) {
  const propertyName = property.name!;
  const title = property.label!;
  const description = property.description?.trim();
  const isEditable = property.editable === true;
  const draft = propertyDrafts[propertyName] ?? "";
  const currentValue = property.value ?? "";
  const propertyDirty = draft !== currentValue;
  const savingThisProperty = isSavingProperty(propertyName);
  const isEditingThis = editingPropertyName === propertyName;
  const inputId = `integration-property-${propertyName}`;

  const exitEdit = () => {
    setEditingPropertyName((current) => (current === propertyName ? null : current));
    setPropertyDrafts((previous) => ({
      ...previous,
      [propertyName]: currentValue,
    }));
  };

  const startEdit = () => setEditingPropertyName(propertyName);

  const trimmedDraft = draft.trim();
  const readonlyHref = trimmedDraft !== "" && isUrl(trimmedDraft) ? trimmedDraft : null;

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-gray-200 pb-4 last:border-b-0 last:pb-0 dark:border-gray-800">
      <Label
        htmlFor={isEditable && isEditingThis ? inputId : undefined}
        className="shrink-0 text-gray-800 dark:text-gray-100"
      >
        {title}
      </Label>
      <PropertyDescriptionTooltip title={title} description={description} />

      <PermissionTooltip
        allowed={canUpdateIntegrations || permissionsLoading}
        message="You don't have permission to update integrations."
      >
        <div className="flex min-w-[min(100%,240px)] flex-1 flex-wrap items-center gap-2">
          {isEditable && isEditingThis ? (
            <PropertyEditingControls
              inputId={inputId}
              propertyName={propertyName}
              draft={draft}
              propertyDirty={propertyDirty}
              savingThisProperty={savingThisProperty}
              canUpdateIntegrations={canUpdateIntegrations}
              settingsMutationBusy={settingsMutationBusy}
              setPropertyDrafts={setPropertyDrafts}
              setEditingPropertyName={setEditingPropertyName}
              exitEdit={exitEdit}
              saveProperty={saveProperty}
            />
          ) : (
            <div className="flex min-w-0 items-center gap-1.5">
              <PropertyReadonlyDisplay draft={draft} trimmedDraft={trimmedDraft} readonlyHref={readonlyHref} />
              {isEditable ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label={`Edit ${title}`}
                  disabled={!canUpdateIntegrations || settingsMutationBusy}
                  onClick={startEdit}
                >
                  <Pencil className="size-4" aria-hidden />
                </Button>
              ) : null}
            </div>
          )}
        </div>
      </PermissionTooltip>
    </div>
  );
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
  const [editingPropertyName, setEditingPropertyName] = useState<string | null>(null);

  if (integrationProperties.length === 0) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No properties for this integration.</p>;
  }

  return (
    <div className="space-y-4 rounded-lg border border-gray-300 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
      {integrationProperties.map((property) => (
        <IntegrationPropertyRow
          key={property.name}
          property={property}
          propertyDrafts={propertyDrafts}
          setPropertyDrafts={setPropertyDrafts}
          editingPropertyName={editingPropertyName}
          setEditingPropertyName={setEditingPropertyName}
          canUpdateIntegrations={canUpdateIntegrations}
          permissionsLoading={permissionsLoading}
          settingsMutationBusy={settingsMutationBusy}
          saveProperty={saveProperty}
          isSavingProperty={isSavingProperty}
        />
      ))}
    </div>
  );
}
