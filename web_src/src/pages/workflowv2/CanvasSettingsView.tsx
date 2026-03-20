import { useEffect, useMemo, useState } from "react";
import { Field, Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";

type ChangeRequestApproverType = "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE";

type CanvasSettingsApprover = {
  type: ChangeRequestApproverType;
  userId?: string;
  roleName?: string;
};

type CanvasSettingsValues = {
  name: string;
  description: string;
  versioningEnabled: boolean;
  changeRequestApprovalConfig?: {
    items?: CanvasSettingsApprover[];
  };
};

type ApproverFieldErrors = {
  type?: string;
  userId?: string;
  roleName?: string;
};

type ApproverValidationResult = {
  formErrors: string[];
  itemErrors: ApproverFieldErrors[];
};

const EMPTY_SELECT_VALUE = "__empty__";

function validateApproverConfig(
  approvers: CanvasSettingsApprover[],
  availableUsers: Array<{ id: string; name: string }>,
  availableRoles: Array<{ name: string; label: string }>,
): ApproverValidationResult {
  if (approvers.length === 0) {
    return {
      formErrors: ["at least one approver is required"],
      itemErrors: [],
    };
  }

  const formErrors: string[] = [];
  const itemErrors: ApproverFieldErrors[] = approvers.map(() => ({}));
  const availableUserIDs = new Set(availableUsers.map((user) => user.id));
  const availableRoleNames = new Set(availableRoles.map((role) => role.name));
  let hasAnyUserApprover = false;
  const seenUsers = new Set<string>();
  const seenRoles = new Set<string>();

  approvers.forEach((approver, index) => {
    if (approver.type === "TYPE_ANYONE") {
      if (hasAnyUserApprover) {
        itemErrors[index].type = "Duplicate any-user approver is not allowed";
      }
      hasAnyUserApprover = true;
      return;
    }

    if (approver.type === "TYPE_USER") {
      const userId = (approver.userId || "").trim();
      if (!userId) {
        itemErrors[index].userId = "User is required";
        return;
      }
      if (!availableUserIDs.has(userId)) {
        itemErrors[index].userId = "Selected user was not found in this organization";
      }
      if (seenUsers.has(userId)) {
        itemErrors[index].userId = "Duplicate user approver is not allowed";
        return;
      }
      seenUsers.add(userId);
      return;
    }

    if (approver.type === "TYPE_ROLE") {
      const roleName = (approver.roleName || "").trim();
      if (!roleName) {
        itemErrors[index].roleName = "Role is required";
        return;
      }
      if (!availableRoleNames.has(roleName)) {
        itemErrors[index].roleName = "Selected role was not found in this organization";
      }
      if (seenRoles.has(roleName)) {
        itemErrors[index].roleName = "Duplicate role approver is not allowed";
        return;
      }
      seenRoles.add(roleName);
      return;
    }

    itemErrors[index].type = "Unsupported approver type";
  });

  return { formErrors, itemErrors };
}

interface CanvasSettingsViewProps {
  initialValues: CanvasSettingsValues;
  canUpdateCanvas: boolean;
  orgVersioningEnabled?: boolean;
  isSaving: boolean;
  availableUsers: Array<{ id: string; name: string }>;
  availableRoles: Array<{ name: string; label: string }>;
  onSave: (values: {
    name: string;
    description: string;
    versioningEnabled?: boolean;
    changeRequestApprovalConfig?: {
      items?: Array<{ type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE"; userId?: string; roleName?: string }>;
    };
  }) => Promise<void>;
}

function normalizeApprovers(items?: CanvasSettingsApprover[]): CanvasSettingsApprover[] {
  const normalized = (items || []).map((item) => ({
    type: item.type,
    userId: item.userId,
    roleName: item.roleName,
  }));
  if (normalized.length > 0) {
    return normalized;
  }

  return [{ type: "TYPE_ANYONE" }];
}

export function CanvasSettingsView({
  initialValues,
  canUpdateCanvas,
  orgVersioningEnabled,
  isSaving,
  availableUsers,
  availableRoles,
  onSave,
}: CanvasSettingsViewProps) {
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [versioningEnabled, setVersioningEnabled] = useState(initialValues.versioningEnabled);
  const [approvers, setApprovers] = useState<CanvasSettingsApprover[]>(
    normalizeApprovers(initialValues.changeRequestApprovalConfig?.items),
  );
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const isVersioningEnforcedByOrganization = orgVersioningEnabled === true;
  const effectiveCanvasVersioningEnabled = isVersioningEnforcedByOrganization ? true : versioningEnabled;
  const isVersioningToggleDisabled = !canUpdateCanvas || isVersioningEnforcedByOrganization;
  const versioningEnforcedTooltip = "Versioning is enabled by your organization settings for all canvases.";

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

  const handleSave = async () => {
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
  };

  const addApprover = () => {
    setApprovers((current) => [...current, { type: "TYPE_ANYONE" }]);
  };

  const updateApproverType = (index: number, type: ChangeRequestApproverType) => {
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
  };

  const updateApproverUser = (index: number, userId: string) => {
    setApprovers((current) =>
      current.map((item, currentIndex) => (currentIndex === index ? { ...item, userId } : item)),
    );
  };

  const updateApproverRole = (index: number, roleName: string) => {
    setApprovers((current) =>
      current.map((item, currentIndex) => (currentIndex === index ? { ...item, roleName } : item)),
    );
  };

  const removeApprover = (index: number) => {
    setApprovers((current) => {
      const next = current.filter((_, currentIndex) => currentIndex !== index);
      return next.length > 0 ? next : [{ type: "TYPE_ANYONE" }];
    });
  };

  const versioningSection = (
    <Fieldset
      className={`rounded-lg border border-gray-300 bg-white p-6 ${isVersioningEnforcedByOrganization ? "opacity-60" : ""}`}
    >
      <div className="flex items-start justify-between gap-6">
        <div>
          <Label className="mb-1 block text-sm font-medium text-gray-700">Canvas Versioning</Label>
          <p className="text-sm text-gray-500">
            Manage canvas edits with drafts and publish flow. When disabled, users edit the live canvas directly.
            {isVersioningEnforcedByOrganization
              ? " Versioning is enabled by your organization settings for all canvases."
              : " This toggle controls versioning for this canvas."}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-500">
            {isVersioningEnforcedByOrganization ? "Enabled" : versioningEnabled ? "Enabled" : "Disabled"}
          </span>
          <Switch
            checked={isVersioningEnforcedByOrganization ? true : versioningEnabled}
            onCheckedChange={setVersioningEnabled}
            disabled={isVersioningToggleDisabled}
            aria-label="Toggle canvas versioning"
          />
        </div>
      </div>
    </Fieldset>
  );

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-6 py-6">
      <Fieldset className="space-y-6 rounded-lg border border-gray-300 bg-white p-6">
        <Field className="space-y-3">
          <Label className="block text-sm font-medium text-gray-700">Canvas Name</Label>
          <Input
            type="text"
            value={name}
            onChange={(event) => setName(event.target.value)}
            disabled={!canUpdateCanvas}
          />
        </Field>

        <Field className="space-y-3">
          <Label className="block text-sm font-medium text-gray-700">Description</Label>
          <Textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            disabled={!canUpdateCanvas}
            rows={4}
          />
        </Field>
      </Fieldset>

      {isVersioningEnforcedByOrganization ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <div aria-disabled="true" className="cursor-not-allowed">
              {versioningSection}
            </div>
          </TooltipTrigger>
          <TooltipContent side="top">{versioningEnforcedTooltip}</TooltipContent>
        </Tooltip>
      ) : (
        versioningSection
      )}

      {effectiveCanvasVersioningEnabled ? (
        <Fieldset className="space-y-4 rounded-lg border border-gray-300 bg-white p-6">
          <div>
            <Label className="mb-1 block text-sm font-medium text-gray-700">Change Request Approvers</Label>
            <p className="text-sm text-gray-500">Define who can approve or reject change requests for this canvas.</p>
          </div>

          {approverValidation.formErrors.map((error) => (
            <p key={error} className="text-xs text-red-600">
              {error}
            </p>
          ))}

          <div className="space-y-3">
            {approvers.map((approver, index) => (
              <div key={`approver-${index}`} className="rounded-md bg-slate-50/70 p-3">
                <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto] md:items-start">
                  <div className="relative pb-4">
                    <Label className="text-xs text-gray-600">Request approval from</Label>
                    <Select
                      value={approver.type}
                      disabled={!canUpdateCanvas}
                      onValueChange={(value) => updateApproverType(index, value as ChangeRequestApproverType)}
                    >
                      <SelectTrigger className="mt-1 h-9 w-full">
                        <SelectValue placeholder="Select approver type" />
                      </SelectTrigger>
                      <SelectContent className="max-h-60">
                        <SelectItem value="TYPE_ANYONE">Any user</SelectItem>
                        <SelectItem value="TYPE_USER">Specific user</SelectItem>
                        <SelectItem value="TYPE_ROLE">Role</SelectItem>
                      </SelectContent>
                    </Select>
                    {approverValidation.itemErrors[index]?.type ? (
                      <p className="pointer-events-none absolute bottom-0 left-0 text-xs text-red-600">
                        {approverValidation.itemErrors[index]?.type}
                      </p>
                    ) : null}
                  </div>

                  {approver.type === "TYPE_USER" ? (
                    <div className="relative pb-4">
                      <Label className="text-xs text-gray-600">User</Label>
                      <Select
                        value={approver.userId || EMPTY_SELECT_VALUE}
                        disabled={!canUpdateCanvas}
                        onValueChange={(value) => updateApproverUser(index, value === EMPTY_SELECT_VALUE ? "" : value)}
                      >
                        <SelectTrigger className="mt-1 h-9 w-full">
                          <SelectValue placeholder="Select a user" />
                        </SelectTrigger>
                        <SelectContent className="max-h-60">
                          <SelectItem value={EMPTY_SELECT_VALUE}>Select a user</SelectItem>
                          {availableUsers.map((user) => (
                            <SelectItem key={user.id} value={user.id}>
                              {user.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {approverValidation.itemErrors[index]?.userId ? (
                        <p className="pointer-events-none absolute bottom-0 left-0 text-xs text-red-600">
                          {approverValidation.itemErrors[index]?.userId}
                        </p>
                      ) : null}
                    </div>
                  ) : approver.type === "TYPE_ROLE" ? (
                    <div className="relative pb-4">
                      <Label className="text-xs text-gray-600">Role</Label>
                      <Select
                        value={approver.roleName || EMPTY_SELECT_VALUE}
                        disabled={!canUpdateCanvas}
                        onValueChange={(value) => updateApproverRole(index, value === EMPTY_SELECT_VALUE ? "" : value)}
                      >
                        <SelectTrigger className="mt-1 h-9 w-full">
                          <SelectValue placeholder="Select a role" />
                        </SelectTrigger>
                        <SelectContent className="max-h-60">
                          <SelectItem value={EMPTY_SELECT_VALUE}>Select a role</SelectItem>
                          {availableRoles.map((role) => (
                            <SelectItem key={role.name} value={role.name}>
                              {role.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {approverValidation.itemErrors[index]?.roleName ? (
                        <p className="pointer-events-none absolute bottom-0 left-0 text-xs text-red-600">
                          {approverValidation.itemErrors[index]?.roleName}
                        </p>
                      ) : null}
                    </div>
                  ) : (
                    <div className="text-xs text-gray-500 md:pb-2">Any authenticated user can approve.</div>
                  )}

                  <div className="flex h-full items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      disabled={!canUpdateCanvas}
                      onClick={() => removeApprover(index)}
                    >
                      Remove
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <Button type="button" variant="secondary" disabled={!canUpdateCanvas} onClick={addApprover}>
            Add Approver
          </Button>
        </Fieldset>
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
  );
}
