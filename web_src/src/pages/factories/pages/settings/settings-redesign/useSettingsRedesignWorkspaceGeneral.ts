import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { factoryListPath, factorySettingsGeneralPathAfterKeyChange } from "../../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../../lib/lastVisitedFactory";
import {
  WORKSPACE_KEY_MAX_LENGTH,
  WORKSPACE_KEY_MIN_LENGTH,
  isValidWorkspaceKey,
  normalizeWorkspaceKey,
} from "../../../lib/workspaceKey";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";

export const WORKSPACE_NAME_MAX_LENGTH = 128;
export const WORKSPACE_DESCRIPTION_MAX_LENGTH = 500;

export type SettingsRedesignWorkspaceGeneral = ReturnType<typeof useSettingsRedesignWorkspaceGeneral>;

export function useSettingsRedesignWorkspaceGeneral() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { account } = useAccount();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const deleteFactory = useDeleteFactory(organizationId);

  const [name, setName] = useState(factory.name ?? "");
  const [description, setDescription] = useState(factory.description ?? "");
  const [key, setKey] = useState(factory.key ?? "");
  const [nameError, setNameError] = useState("");
  const [keyError, setKeyError] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);

  useEffect(() => {
    setName(factory.name ?? "");
    setDescription(factory.description ?? "");
    setKey(factory.key ?? "");
    setNameError("");
    setKeyError("");
  }, [factory.name, factory.description, factory.key]);

  const canUpdate = canAct("factories", "update");
  const canDelete = canAct("factories", "delete");

  const saveDetails = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }
    if (!isValidWorkspaceKey(key)) {
      setKeyError(`Use ${WORKSPACE_KEY_MIN_LENGTH} to ${WORKSPACE_KEY_MAX_LENGTH} uppercase letters.`);
      return;
    }
    try {
      const nextSettingsPath = factorySettingsGeneralPathAfterKeyChange(organizationId, factory.key ?? "", key);
      await updateFactory.mutateAsync({
        name: trimmedName,
        description: description.trim(),
        key: key !== (factory.key ?? "") ? key : undefined,
      });
      showSuccessToast("Workspace updated.");
      if (nextSettingsPath) {
        navigate(nextSettingsPath, { replace: true });
      }
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to update workspace");
      if (message.toLowerCase().includes("workspace key")) {
        setKeyError(message);
        return;
      }
      showErrorToast(message);
    }
  };

  const handleDelete = async () => {
    try {
      await deleteFactory.mutateAsync(factoryId);
      clearLastVisitedFactory(account?.id ?? "", organizationId, factoryId);
      showSuccessToast("Workspace deleted.");
      navigate(factoryListPath(organizationId));
    } catch {
      showErrorToast("Failed to delete workspace.");
      throw new Error("Failed to delete workspace");
    }
  };

  return {
    factory,
    updateFactory,
    deleteFactory,
    name,
    setName,
    description,
    setDescription,
    key,
    setKey: (next: string) => setKey(normalizeWorkspaceKey(next)),
    nameError,
    clearNameError: () => setNameError(""),
    keyError,
    clearKeyError: () => setKeyError(""),
    deleteOpen,
    setDeleteOpen,
    canUpdate,
    canDelete,
    permissionsLoading,
    isDirty:
      name.trim() !== (factory.name ?? "") ||
      key !== (factory.key ?? "") ||
      description.trim() !== (factory.description ?? ""),
    saveDetails,
    handleDelete,
  };
}
