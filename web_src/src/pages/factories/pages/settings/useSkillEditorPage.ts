import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";

import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,62}$/;
const RESERVED_NAME = "superplane";

export function validateSkillName(name: string): string {
  if (!name) {
    return AGENT_RESOURCES_COPY.nameRequired;
  }
  if (name === RESERVED_NAME) {
    return AGENT_RESOURCES_COPY.nameReserved;
  }
  if (!NAME_PATTERN.test(name)) {
    return AGENT_RESOURCES_COPY.nameInvalid;
  }
  return "";
}

export function skillFrontmatterName(markdown: string): string | undefined {
  const match = markdown.match(/^---\s*\n([\s\S]*?)\n---/);
  if (!match) {
    return undefined;
  }
  const nameLine = match[1]
    .split("\n")
    .map((line) => line.trim())
    .find((line) => line.startsWith("name:"));
  if (!nameLine) {
    return undefined;
  }
  return nameLine
    .slice("name:".length)
    .trim()
    .replace(/^["']|["']$/g, "");
}

export function setSkillFrontmatterName(markdown: string, name: string): string {
  const match = markdown.match(/^---\s*\n([\s\S]*?)\n---(\n[\s\S]*)?$/);
  if (!match) {
    return `---\nname: ${name}\n---\n${markdown}`;
  }
  const body = match[2] ?? "\n";
  const lines = match[1].split("\n");
  const nameIndex = lines.findIndex((line) => line.trim().startsWith("name:"));
  if (nameIndex >= 0) {
    lines[nameIndex] = `name: ${name}`;
  } else {
    lines.unshift(`name: ${name}`);
  }
  return `---\n${lines.join("\n")}\n---${body.startsWith("\n") ? body : `\n${body}`}`;
}

export function useSkillEditorPage() {
  const { resourceId } = useParams<{ resourceId?: string }>();
  const isCreate = !resourceId || resourceId === "new";
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const navigate = useNavigate();
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const resource = isCreate ? undefined : skills.data?.find((entry) => entry.id === resourceId);
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const [name, setName] = useState("");
  const [markdown, setMarkdown] = useState("");
  const [nameError, setNameError] = useState("");
  const [markdownError, setMarkdownError] = useState("");
  const [pendingDelete, setPendingDelete] = useState(false);
  const listPath = factorySettingsSectionPath(organizationId, factory.key ?? "", "workspace", "skills");

  useEffect(() => {
    if (isCreate) {
      setName("");
      setMarkdown("---\nname: \ndescription: \n---\n\n");
      return;
    }
    if (!resource) {
      return;
    }
    setName(resource.name ?? "");
    setMarkdown(resource.markdown ?? "");
  }, [isCreate, resource]);

  const trimmedName = name.trim().toLowerCase();
  const frontmatterName = useMemo(() => skillFrontmatterName(markdown), [markdown]);
  const actions = {
    handleSave: () =>
      saveSkill({
        trimmedName,
        markdown,
        resourceId: resource?.id,
        listPath,
        navigate,
        setNameError,
        setMarkdownError,
        createResource,
        updateResource,
      }),
    confirmDelete: () => deleteSkill({ resourceId: resource?.id, listPath, navigate, deleteResource }),
  };

  return {
    isCreate,
    factory,
    canUpdate,
    resource,
    skillsLoading: skills.isLoading,
    name,
    markdown,
    nameError,
    markdownError,
    pendingDelete,
    listPath,
    trimmedName,
    command: trimmedName ? `/${trimmedName}` : "/",
    frontmatterMismatch: Boolean(trimmedName && frontmatterName && frontmatterName !== trimmedName),
    isSaving: createResource.isPending || updateResource.isPending,
    isDeleting: deleteResource.isPending,
    setName,
    setMarkdown,
    setPendingDelete,
    navigateToList: () => navigate(listPath),
    ...actions,
  };
}

async function saveSkill({
  trimmedName,
  markdown,
  resourceId,
  listPath,
  navigate,
  setNameError,
  setMarkdownError,
  createResource,
  updateResource,
}: {
  trimmedName: string;
  markdown: string;
  resourceId?: string;
  listPath: string;
  navigate: ReturnType<typeof useNavigate>;
  setNameError: (value: string) => void;
  setMarkdownError: (value: string) => void;
  createResource: ReturnType<typeof useCreateFactoryAgentResource>;
  updateResource: ReturnType<typeof useUpdateFactoryAgentResource>;
}) {
  const nextNameError = validateSkillName(trimmedName);
  const nextMarkdownError = markdown.trim() ? "" : AGENT_RESOURCES_COPY.markdownRequired;
  setNameError(nextNameError);
  setMarkdownError(nextMarkdownError);
  if (nextNameError || nextMarkdownError) {
    return;
  }
  try {
    if (resourceId) {
      await updateResource.mutateAsync({ resourceId, name: trimmedName, markdown: markdown.trim() });
      showSuccessToast(AGENT_RESOURCES_COPY.skillUpdated);
    } else {
      await createResource.mutateAsync({
        kind: "KIND_SKILL",
        name: trimmedName,
        enabled: true,
        markdown: markdown.trim(),
      });
      showSuccessToast(AGENT_RESOURCES_COPY.skillCreated);
    }
    navigate(listPath);
  } catch (error) {
    showErrorToast(
      getApiErrorMessage(
        error,
        resourceId ? AGENT_RESOURCES_COPY.skillUpdateFailed : AGENT_RESOURCES_COPY.skillCreateFailed,
      ),
    );
  }
}

async function deleteSkill({
  resourceId,
  listPath,
  navigate,
  deleteResource,
}: {
  resourceId?: string;
  listPath: string;
  navigate: ReturnType<typeof useNavigate>;
  deleteResource: ReturnType<typeof useDeleteFactoryAgentResource>;
}) {
  if (!resourceId) {
    return;
  }
  try {
    await deleteResource.mutateAsync(resourceId);
    showSuccessToast(AGENT_RESOURCES_COPY.skillDeleted);
    navigate(listPath);
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.skillDeleteFailed));
    throw error;
  }
}
