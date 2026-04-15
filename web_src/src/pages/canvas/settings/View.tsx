import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { ArrowLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { normalizeApprovers, validateApproverConfig } from "./approverUtils";
import { IdentityFields } from "./IdentityFields";
import type { ChangeRequestApproverType, SettingsApprover, SettingsViewProps } from "./types";
import { ChangeManagementFieldset } from "./ChangeManagementFieldset";
import { isChangeManagementSettingsEnabled } from "@/lib/env";

export function SettingsView({
  initialValues,
  canUpdateCanvas,
  orgChangeManagementEnabled,
  isSaving,
  availableUsers,
  availableRoles,
  onSave,
  onBackToCanvas,
}: SettingsViewProps) {
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [changeManagementEnabled, setChangeManagementEnabled] = useState(initialValues.changeManagementEnabled);
  const [approvers, setApprovers] = useState<SettingsApprover[]>(
    normalizeApprovers(initialValues.changeRequestApprovalConfig?.items),
  );
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const isChangeManagementEnforcedByOrganization = orgChangeManagementEnabled === true;
  const effectiveChangeManagementEnabled = isChangeManagementEnforcedByOrganization ? true : changeManagementEnabled;
  const isChangeManagementToggleDisabled = !canUpdateCanvas || isChangeManagementEnforcedByOrganization;

  useEffect(() => {
    setName(initialValues.name);
    setDescription(initialValues.description);
    setChangeManagementEnabled(isChangeManagementEnforcedByOrganization ? true : initialValues.changeManagementEnabled);
    setApprovers(normalizeApprovers(initialValues.changeRequestApprovalConfig?.items));
  }, [initialValues, isChangeManagementEnforcedByOrganization]);

  const normalizedInitialApprovers = useMemo(
    () => normalizeApprovers(initialValues.changeRequestApprovalConfig?.items),
    [initialValues.changeRequestApprovalConfig?.items],
  );

  const hasChanges = useMemo(() => {
    return (
      name !== initialValues.name ||
      description !== initialValues.description ||
      effectiveChangeManagementEnabled !== initialValues.changeManagementEnabled ||
      JSON.stringify(approvers) !== JSON.stringify(normalizedInitialApprovers)
    );
  }, [
    description,
    effectiveChangeManagementEnabled,
    initialValues.changeManagementEnabled,
    initialValues.description,
    initialValues.name,
    name,
    approvers,
    normalizedInitialApprovers,
  ]);

  const approverValidation = useMemo(() => {
    if (!effectiveChangeManagementEnabled) {
      return { formErrors: [], itemErrors: [] };
    }
    return validateApproverConfig(approvers, availableUsers, availableRoles);
  }, [approvers, availableRoles, availableUsers, effectiveChangeManagementEnabled]);

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
        changeManagementEnabled: isChangeManagementEnforcedByOrganization ? undefined : changeManagementEnabled,
        changeRequestApprovalConfig: effectiveChangeManagementEnabled
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
    effectiveChangeManagementEnabled,
    hasApproverValidationErrors,
    isChangeManagementEnforcedByOrganization,
    name,
    onSave,
    changeManagementEnabled,
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
        {isChangeManagementSettingsEnabled() ? (
          <ChangeManagementFieldset
            isChangeManagementEnforcedByOrganization={isChangeManagementEnforcedByOrganization}
            changeManagementEnabled={changeManagementEnabled}
            onChangeManagementEnabledChange={setChangeManagementEnabled}
            isChangeManagementToggleDisabled={isChangeManagementToggleDisabled}
            effectiveChangeManagementEnabled={effectiveChangeManagementEnabled}
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
        ) : null}
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
