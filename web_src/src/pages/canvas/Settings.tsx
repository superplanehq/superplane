import type { CanvasesCanvas } from "@/api-client";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Field, Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useCanvas, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useOrganization, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

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
  /** When set, shows a control above the form to return to the canvas editor. */
  onBackToCanvas?: () => void;
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

  return [{ type: "TYPE_USER", userId: "" }];
}

export function CanvasSettingsView({
  initialValues,
  canUpdateCanvas,
  orgVersioningEnabled,
  isSaving,
  availableUsers,
  availableRoles,
  onSave,
  onBackToCanvas,
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
  const hasEveryoneApprover = useMemo(() => approvers.some((a) => a.type === "TYPE_ANYONE"), [approvers]);

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
    setApprovers((current) => [...current, { type: "TYPE_USER", userId: "" }]);
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
      return next.length > 0 ? next : [{ type: "TYPE_USER", userId: "" }];
    });
  };

  const versioningContent = (
    <div className="flex items-start justify-between gap-6">
      <div>
        <Label htmlFor="canvas-versioning-switch" className="mb-1 block text-sm font-medium text-gray-700">
          Canvas Versioning
        </Label>
        <p className="text-[13px] text-gray-500">
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
          id="canvas-versioning-switch"
          checked={isVersioningEnforcedByOrganization ? true : versioningEnabled}
          onCheckedChange={setVersioningEnabled}
          disabled={isVersioningToggleDisabled}
          aria-label="Toggle canvas versioning"
        />
      </div>
    </div>
  );

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
        <Fieldset className="space-y-6 rounded-lg border border-slate-950/15 bg-white p-6">
          <Field className="space-y-3">
            <Label htmlFor="canvas-settings-name-input" className="block text-sm font-medium text-gray-700">
              Canvas Name
            </Label>
            <Input
              id="canvas-settings-name-input"
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={!canUpdateCanvas}
            />
          </Field>

          <Field className="space-y-3">
            <Label htmlFor="canvas-settings-description-input" className="block text-sm font-medium text-gray-700">
              Description
            </Label>
            <Textarea
              id="canvas-settings-description-input"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              disabled={!canUpdateCanvas}
              rows={4}
              placeholder="Describe canvas…"
            />
          </Field>
        </Fieldset>

        <Fieldset className="rounded-lg border border-slate-950/15 bg-white p-6">
          {isVersioningEnforcedByOrganization ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="cursor-not-allowed opacity-60">{versioningContent}</div>
              </TooltipTrigger>
              <TooltipContent side="top">{versioningEnforcedTooltip}</TooltipContent>
            </Tooltip>
          ) : (
            versioningContent
          )}
          {effectiveCanvasVersioningEnabled ? (
            <div className="mt-6 border-t border-slate-950/10 pt-6 space-y-4">
              <div>
                <p className="mb-1 block text-sm font-medium text-gray-700">Who can approve changes</p>
                <p className="text-[13px] text-gray-500">
                  Define who can approve or reject change requests for this canvas.
                </p>
              </div>

              {approverValidation.formErrors.map((error) => (
                <p key={error} className="text-xs text-red-600">
                  {error}
                </p>
              ))}

              <div className="space-y-3">
                {approvers.map((approver, index) => (
                  <div key={`approver-${index}`} className="border-b border-slate-950/10 py-3">
                    <div className="grid gap-3 md:grid-cols-[auto_1fr_auto] md:items-start">
                      <div className="w-full md:w-[12rem] md:justify-self-start">
                        <Select
                          value={approver.type}
                          disabled={!canUpdateCanvas}
                          onValueChange={(value) => updateApproverType(index, value as ChangeRequestApproverType)}
                        >
                          <SelectTrigger className="h-9 w-full" aria-label="Request approval from">
                            <SelectValue placeholder="Select approver type" />
                          </SelectTrigger>
                          <SelectContent className="max-h-60">
                            <SelectItem value="TYPE_ANYONE">Everyone</SelectItem>
                            <SelectItem value="TYPE_USER">Specific user</SelectItem>
                            <SelectItem value="TYPE_ROLE">Role</SelectItem>
                          </SelectContent>
                        </Select>
                        {approverValidation.itemErrors[index]?.type ? (
                          <p className="mt-2 text-xs text-red-600">{approverValidation.itemErrors[index]?.type}</p>
                        ) : null}
                      </div>

                      {approver.type === "TYPE_USER" ? (
                        <div>
                          <Select
                            value={approver.userId || EMPTY_SELECT_VALUE}
                            disabled={!canUpdateCanvas}
                            onValueChange={(value) =>
                              updateApproverUser(index, value === EMPTY_SELECT_VALUE ? "" : value)
                            }
                          >
                            <SelectTrigger className="h-9 w-full" aria-label="User">
                              <SelectValue placeholder="Select a user…" />
                            </SelectTrigger>
                            <SelectContent className="max-h-60">
                              <SelectItem value={EMPTY_SELECT_VALUE}>Select a user…</SelectItem>
                              {availableUsers.map((user) => (
                                <SelectItem key={user.id} value={user.id}>
                                  {user.name}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          {approverValidation.itemErrors[index]?.userId ? (
                            <p className="mt-2 text-xs text-red-600">{approverValidation.itemErrors[index]?.userId}</p>
                          ) : null}
                        </div>
                      ) : approver.type === "TYPE_ROLE" ? (
                        <div>
                          <Select
                            value={approver.roleName || EMPTY_SELECT_VALUE}
                            disabled={!canUpdateCanvas}
                            onValueChange={(value) =>
                              updateApproverRole(index, value === EMPTY_SELECT_VALUE ? "" : value)
                            }
                          >
                            <SelectTrigger className="h-9 w-full" aria-label="Role">
                              <SelectValue placeholder="Select a role…" />
                            </SelectTrigger>
                            <SelectContent className="max-h-60">
                              <SelectItem value={EMPTY_SELECT_VALUE}>Select a role…</SelectItem>
                              {availableRoles.map((role) => (
                                <SelectItem key={role.name} value={role.name}>
                                  {role.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          {approverValidation.itemErrors[index]?.roleName ? (
                            <p className="mt-2 text-xs text-red-600">
                              {approverValidation.itemErrors[index]?.roleName}
                            </p>
                          ) : null}
                        </div>
                      ) : (
                        <div className="self-center text-xs text-gray-500">Any authenticated user can approve.</div>
                      )}

                      <div className="flex h-full items-start gap-2">
                        <Button
                          type="button"
                          variant="outline"
                          disabled={!canUpdateCanvas || approvers.length <= 1}
                          onClick={() => removeApprover(index)}
                        >
                          Remove
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              <Button
                type="button"
                variant="outline"
                disabled={!canUpdateCanvas || hasEveryoneApprover}
                onClick={addApprover}
              >
                Add Approver
              </Button>
            </div>
          ) : null}
        </Fieldset>

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

function buildSettingsInitialValues(canvas: CanvasesCanvas | undefined) {
  return {
    name: canvas?.metadata?.name || "",
    description: canvas?.metadata?.description || "",
    versioningEnabled: canvas?.metadata?.versioningEnabled ?? false,
    changeRequestApprovalConfig: {
      items: (canvas?.metadata?.changeRequestApprovalConfig?.items || [])
        .map((item) => {
          if (!item.type || (item.type !== "TYPE_ANYONE" && item.type !== "TYPE_USER" && item.type !== "TYPE_ROLE")) {
            return null;
          }
          return {
            type: item.type,
            userId: item.userId,
            roleName: item.roleName,
          };
        })
        .filter(
          (
            item,
          ): item is {
            type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE";
            userId: string | undefined;
            roleName: string | undefined;
          } => !!item,
        ),
    },
  };
}

export function CanvasSettingsPage() {
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();
  const navigate = useNavigate();
  const { canAct } = usePermissions();
  const canReadOrg = canAct("org", "read");
  const canUpdateCanvas = canAct("canvases", "update");

  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId || "", canvasId || "");

  const { data: organization } = useOrganization(organizationId || "", !!organizationId && canReadOrg);
  const isOrgVersioningEnabled = organization?.metadata?.versioningEnabled;

  const { data: organizationUsers = [] } = useOrganizationUsers(organizationId || "");
  const { data: organizationRoles = [] } = useOrganizationRoles(organizationId || "");

  const updateCanvasMutation = useUpdateCanvas(organizationId || "", canvasId || "");

  const isTemplate = canvas?.metadata?.isTemplate ?? false;
  const baseCanvasPath = useMemo(() => {
    if (!organizationId || !canvasId) {
      return "";
    }
    if (canvas?.metadata?.isTemplate) {
      return `/${organizationId}/templates/${canvasId}`;
    }
    return `/${organizationId}/canvases/${canvasId}`;
  }, [organizationId, canvasId, canvas?.metadata?.isTemplate]);

  usePageTitle([canvas?.metadata?.name ? `${canvas.metadata.name} · Settings` : "Canvas settings"]);

  const initialValues = useMemo(() => buildSettingsInitialValues(canvas), [canvas]);

  const approverUsers = useMemo(
    () =>
      organizationUsers
        .map((user) => {
          const id = user.metadata?.id || "";
          if (!id) {
            return null;
          }
          return {
            id,
            name: user.spec?.displayName || user.metadata?.email || id,
          };
        })
        .filter((item): item is { id: string; name: string } => !!item),
    [organizationUsers],
  );

  const approverRoles = useMemo(
    () =>
      organizationRoles
        .map((role) => {
          const name = role.metadata?.name || "";
          if (!name) {
            return null;
          }
          return {
            name,
            label: role.spec?.displayName || name,
          };
        })
        .filter((item): item is { name: string; label: string } => !!item),
    [organizationRoles],
  );

  const handleSave = useCallback(
    async (values: {
      name: string;
      description: string;
      versioningEnabled?: boolean;
      changeRequestApprovalConfig?: {
        items?: Array<{ type: "TYPE_ANYONE" | "TYPE_USER" | "TYPE_ROLE"; userId?: string; roleName?: string }>;
      };
    }) => {
      if (!canvasId) {
        return;
      }
      await updateCanvasMutation.mutateAsync({
        name: values.name,
        description: values.description,
        versioningEnabled: values.versioningEnabled,
        changeRequestApprovalConfig: values.changeRequestApprovalConfig,
      });
    },
    [canvasId, updateCanvasMutation],
  );

  if (!organizationId || !canvasId) {
    return (
      <div className="flex h-full items-center justify-center bg-slate-100 text-sm text-slate-600">
        Missing organization or canvas.
      </div>
    );
  }

  if (canvasLoading) {
    return (
      <div className="flex h-full flex-col bg-slate-100">
        <header className="flex h-11 items-center border-b border-slate-950/15 bg-white px-3 sm:px-4">
          <OrganizationMenuButton organizationId={organizationId} />
        </header>
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-slate-400" aria-label="Loading" />
        </div>
      </div>
    );
  }

  if (canvasError || !canvas) {
    return (
      <div className="flex h-full flex-col bg-slate-100">
        <header className="flex h-11 items-center border-b border-slate-950/15 bg-white px-3 sm:px-4">
          <OrganizationMenuButton organizationId={organizationId} />
        </header>
        <div className="flex flex-1 flex-col items-center justify-center gap-3 px-4 text-center">
          <p className="text-sm text-slate-600">This canvas could not be loaded.</p>
          <Link
            to={baseCanvasPath || `/${organizationId}`}
            className="text-sm font-medium text-sky-700 hover:underline"
          >
            Back to canvas
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-slate-100">
      <header className="relative flex h-11 shrink-0 items-center border-b border-slate-950/15 bg-white px-3 sm:px-4">
        <div className="relative z-10 flex min-w-0 shrink-0 items-center">
          <OrganizationMenuButton organizationId={organizationId} />
        </div>
        <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
          <span className="truncate text-center text-sm font-medium text-slate-900">
            {canvas.metadata?.name || "Canvas"} · Settings
          </span>
        </div>
        <div className="relative z-10 ml-auto w-9 shrink-0" aria-hidden />
      </header>
      <div className="min-h-0 flex-1 overflow-auto">
        <CanvasSettingsView
          initialValues={initialValues}
          canUpdateCanvas={canUpdateCanvas && !isTemplate}
          orgVersioningEnabled={isOrgVersioningEnabled}
          isSaving={updateCanvasMutation.isPending}
          availableUsers={approverUsers}
          availableRoles={approverRoles}
          onSave={handleSave}
          onBackToCanvas={() => navigate(baseCanvasPath)}
        />
      </div>
    </div>
  );
}
