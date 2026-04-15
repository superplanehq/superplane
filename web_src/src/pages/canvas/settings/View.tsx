import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { ArrowLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { normalizeApprovers, validateApproverConfig } from "./approverUtils";
import { IdentityFields } from "./IdentityFields";
import type { ChangeRequestApproverType, SettingsApprover, SettingsViewProps } from "./types";
import { VersioningFieldset } from "./VersioningFieldset";

export function SettingsView({
  initialValues,
  canUpdateCanvas,
  orgVersioningEnabled,
  isSaving,
  availableUsers,
  availableRoles,
  onSave,
  onBackToCanvas,
}: SettingsViewProps) {
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [versioningEnabled, setVersioningEnabled] = useState(initialValues.versioningEnabled);
  const [approvers, setApprovers] = useState<SettingsApprover[]>(
    normalizeApprovers(initialValues.changeRequestApprovalConfig?.items),
  );
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const isVersioningEnforcedByOrganization = orgVersioningEnabled === true;
  const effectiveCanvasVersioningEnabled = isVersioningEnforcedByOrganization ? true : versioningEnabled;
  const isVersioningToggleDisabled = !canUpdateCanvas || isVersioningEnforcedByOrganization;

  useEffect(() => {
    setName(initialValues.name);
    setDescription(initialValues.description);
    setVersioningEnabled(isVersioningEnforcedByOrganization ? true : initialValues.versioningEnabled);
    setApprovers(normalizeApprovers(initialValues.changeRequestApprovalConfig?.items));
  }, [initialValues, isVersioningEnforcedByOrganization]);

  const normalizedInitialApprovers = useMemo(
    () => normalizeApprovers(initialValues.changeRequestApprovalConfig?.items),
    [initialValues.changeRequestApprovalConfig?.items],
  );

  const hasChanges = useMemo(() => {
    return (
      name !== initialValues.name ||
      description !== initialValues.description ||
      effectiveCanvasVersioningEnabled !== initialValues.versioningEnabled ||
      JSON.stringify(approvers) !== JSON.stringify(normalizedInitialApprovers)
    );
  }, [
    description,
    initialValues.versioningEnabled,
    initialValues.description,
    initialValues.name,
    isVersioningEnforcedByOrganization,
    name,
    approvers,
    normalizedInitialApprovers,
    versioningEnabled,
  ]);

  const approverValidation = useMemo(() => {
    if (!effectiveCanvasVersioningEnabled) {
      return { formErrors: [], itemErrors: [] };
    }
    return validateApproverConfig(approvers, availableUsers, availableRoles);
  }, [approvers, availableRoles, availableUsers, effectiveCanvasVersioningEnabled]);

  const hasApproverValidationErrors = useMemo(
    () =>
      approverValidation.formErrors.length > 0 ||
      approverValidation.itemErrors.some((item) => !!item.type || !!item.userId || !!item.roleName),
    [approverValidation.formErrors.length, approverValidation.itemErrors],
  );

  const hasEveryoneApprover = useMemo(() => approvers.some((a) => a.type === "TYPE_ANYONE"), [approvers]);

  const handleSave = useCallback(async () => {
    if (!canUpdateCanvas) {
      return;
    }

    setSaveMessage(null);
    if (hasApproverValidationErrors) {
      return;
    }

    try {
      await onSave({
        name,
        description,
        versioningEnabled: isVersioningEnforcedByOrganization ? undefined : versioningEnabled,
        changeRequestApprovalConfig: effectiveCanvasVersioningEnabled
          ? {
              items: normalizeApprovers(approvers),
            }
          : undefined,
      });
      setSaveMessage("Canvas updated successfully");
      setTimeout(() => setSaveMessage(null), 3000);
    } catch (error) {
      const responseMessage = (error as { response?: { data?: { message?: string } } })?.response?.data?.message;
      const errorMessage = responseMessage || (error as { message?: string })?.message || "Failed to update canvas";
      setSaveMessage(errorMessage);
      setTimeout(() => setSaveMessage(null), 3000);
    }
  }, [
    approvers,
    canUpdateCanvas,
    description,
    effectiveCanvasVersioningEnabled,
    hasApproverValidationErrors,
    isVersioningEnforcedByOrganization,
    name,
    onSave,
    versioningEnabled,
  ]);

  const addApprover = useCallback(() => {
    setApprovers((current) => [...current, { type: "TYPE_USER", userId: "" }]);
  }, []);

  const updateApproverType = useCallback((index: number, type: ChangeRequestApproverType) => {
    setApprovers((current) =>
      current.map((item, currentIndex) => {
        if (currentIndex !== index) {
          return item;
        }

        if (type === "TYPE_USER") {
          return { type, userId: item.userId || "" };
        }
        if (type === "TYPE_ROLE") {
          return { type, roleName: item.roleName || "" };
        }

        return { type };
      }),
    );
  }, []);

  const updateApproverUser = useCallback((index: number, userId: string) => {
    setApprovers((current) =>
      current.map((item, currentIndex) => (currentIndex === index ? { ...item, userId } : item)),
    );
  }, []);

  const updateApproverRole = useCallback((index: number, roleName: string) => {
    setApprovers((current) =>
      current.map((item, currentIndex) => (currentIndex === index ? { ...item, roleName } : item)),
    );
  }, []);

  const removeApprover = useCallback((index: number) => {
    setApprovers((current) => {
      const next = current.filter((_, currentIndex) => currentIndex !== index);
      return next.length > 0 ? next : [{ type: "TYPE_USER", userId: "" }];
    });
  }, []);

  return (
    <div className="px-4 py-6">
      <div className="mx-auto w-full max-w-3xl space-y-6">
        {onBackToCanvas ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="-ml-2 gap-1 px-2 text-slate-600 hover:text-slate-900"
            onClick={onBackToCanvas}
          >
            <ArrowLeft className="h-4 w-4 shrink-0" aria-hidden />
            Back to canvas
          </Button>
        ) : null}
        <IdentityFields
          name={name}
          description={description}
          onNameChange={setName}
          onDescriptionChange={setDescription}
          canUpdateCanvas={canUpdateCanvas}
        />
        <VersioningFieldset
          isVersioningEnforcedByOrganization={isVersioningEnforcedByOrganization}
          versioningEnabled={versioningEnabled}
          onVersioningEnabledChange={setVersioningEnabled}
          isVersioningToggleDisabled={isVersioningToggleDisabled}
          effectiveCanvasVersioningEnabled={effectiveCanvasVersioningEnabled}
          approvers={approvers}
          canUpdateCanvas={canUpdateCanvas}
          availableUsers={availableUsers}
          availableRoles={availableRoles}
          approverValidation={approverValidation}
          hasEveryoneApprover={hasEveryoneApprover}
          onApproverTypeChange={updateApproverType}
          onApproverUserChange={updateApproverUser}
          onApproverRoleChange={updateApproverRole}
          onRemoveApprover={removeApprover}
          onAddApprover={addApprover}
        />
        <div className="flex items-center gap-4">
          <LoadingButton
            type="button"
            onClick={handleSave}
            disabled={!canUpdateCanvas || !hasChanges || hasApproverValidationErrors}
            loading={isSaving}
            loadingText="Saving..."
          >
            Save Changes
          </LoadingButton>
          {saveMessage ? (
            <span className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600" : "text-red-600"}`}>
              {saveMessage}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}
