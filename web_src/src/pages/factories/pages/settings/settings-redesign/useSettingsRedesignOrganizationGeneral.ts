import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { usePermissions } from "@/contexts/usePermissions";
import { useDeleteOrganization, useOrganization, useUpdateOrganization } from "@/hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { organizationSlugValidationMessage, validateOrganizationSlug } from "@/lib/organizationSlug";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";

export const ORGANIZATION_NAME_MAX_LENGTH = 128;

export function useSettingsRedesignOrganizationGeneral() {
  const { organizationId } = useFactorySettingsLayout();
  const navigate = useNavigate();
  const location = useLocation();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationId);
  const updateOrganization = useUpdateOrganization(organizationId);
  const deleteOrganization = useDeleteOrganization(organizationId);

  const currentName = organization?.metadata?.name || "";
  const currentSlug = organization?.metadata?.slug || "";
  const currentDescription = organization?.metadata?.description || "";

  const [name, setName] = useState(currentName);
  const [slug, setSlug] = useState(currentSlug);
  const [description, setDescription] = useState(currentDescription);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    setName(currentName);
    setSlug(currentSlug);
    setDescription(currentDescription);
    setSlugError(null);
  }, [currentName, currentSlug, currentDescription]);

  const canUpdate = canAct("org", "update");
  const canDelete = canAct("org", "delete");
  const trimmedName = name.trim();
  const trimmedSlug = slug.trim();
  const isDirty =
    trimmedName !== currentName || trimmedSlug !== currentSlug || description.trim() !== currentDescription;

  const saveDetails = async () => {
    if (!canUpdate || !isDirty) return;
    if (!trimmedName) {
      showErrorToast("Name is required.");
      return;
    }
    if (trimmedSlug !== currentSlug) {
      const validationError = validateOrganizationSlug(trimmedSlug);
      if (validationError) {
        setSlugError(organizationSlugValidationMessage(validationError));
        return;
      }
    }
    setSlugError(null);
    try {
      await updateOrganization.mutateAsync({
        name: trimmedName,
        description: description.trim(),
        slug: trimmedSlug !== currentSlug ? trimmedSlug : undefined,
      });
      showSuccessToast("Organization updated.");
      if (organizationId === currentSlug && trimmedSlug !== currentSlug) {
        const newPath = location.pathname.replace(`/${organizationId}/`, `/${trimmedSlug}/`);
        navigate(`${newPath}${location.search}`, { replace: true });
      }
    } catch (error) {
      const message = getApiErrorMessage(error, "Failed to update organization");
      setSlugError(message);
      showErrorToast(message);
    }
  };

  const handleDelete = async () => {
    if (!canDelete) return;
    try {
      setDeleteError(null);
      await deleteOrganization.mutateAsync();
      window.location.href = "/";
    } catch {
      setDeleteError("Failed to delete organization.");
    }
  };

  return {
    organizationId,
    organization,
    name,
    setName,
    slug,
    setSlug,
    description,
    setDescription,
    slugError,
    clearSlugError: () => setSlugError(null),
    deleteError,
    clearDeleteError: () => setDeleteError(null),
    canUpdate,
    canDelete,
    permissionsLoading,
    isDirty,
    updateOrganization,
    deleteOrganization,
    saveDetails,
    handleDelete,
  };
}
