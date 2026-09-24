import { useEffect, useState } from "react";
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
import { sanitizeSkillCommandName, setSkillFrontmatterFields, skillDisplayTitle } from "./skillFrontmatter";

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,62}$/;
const RESERVED_NAME = "superplane";
const EMPTY_SKILL_MARKDOWN = "---\nname: \ntitle: \ndescription: \n---\n\n";

export { sanitizeSkillCommandName };

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
      setMarkdown(EMPTY_SKILL_MARKDOWN);
      return;
    }
    if (!resource) {
      return;
    }
    setName(skillDisplayTitle(resource));
    setMarkdown(resource.markdown ?? "");
  }, [isCreate, resource]);

  const commandName = sanitizeSkillCommandName(name);
  const taken = Boolean(
    commandName && skills.data?.some((entry) => entry.id !== resource?.id && entry.name === commandName),
  );

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
    commandName,
    command: commandName ? `/${commandName}` : "/",
    isSaving: createResource.isPending || updateResource.isPending,
    isDeleting: deleteResource.isPending,
    setName: (value: string) => setNameAndFrontmatter(value, setName, setMarkdown),
    setMarkdown,
    setPendingDelete,
    navigateToList: () => navigate(listPath),
    handleSave: () =>
      saveSkill({
        commandName,
        title: name.trim(),
        markdown,
        taken,
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
}

function setNameAndFrontmatter(
  value: string,
  setName: (value: string) => void,
  setMarkdown: (value: string | ((current: string) => string)) => void,
) {
  setName(value);
  const slug = sanitizeSkillCommandName(value);
  setMarkdown((current) => setSkillFrontmatterFields(current, { name: slug, title: value }));
}

async function saveSkill({
  commandName,
  title,
  markdown,
  taken,
  resourceId,
  listPath,
  navigate,
  setNameError,
  setMarkdownError,
  createResource,
  updateResource,
}: {
  commandName: string;
  title: string;
  markdown: string;
  taken: boolean;
  resourceId?: string;
  listPath: string;
  navigate: ReturnType<typeof useNavigate>;
  setNameError: (value: string) => void;
  setMarkdownError: (value: string) => void;
  createResource: ReturnType<typeof useCreateFactoryAgentResource>;
  updateResource: ReturnType<typeof useUpdateFactoryAgentResource>;
}) {
  const nextNameError = taken ? AGENT_RESOURCES_COPY.nameTaken : validateSkillName(commandName);
  const nextMarkdownError = markdown.trim() ? "" : AGENT_RESOURCES_COPY.markdownRequired;
  setNameError(nextNameError);
  setMarkdownError(nextMarkdownError);
  if (nextNameError || nextMarkdownError) {
    return;
  }
  const nextMarkdown = setSkillFrontmatterFields(markdown.trim(), { name: commandName, title }).trim();
  try {
    if (resourceId) {
      await updateResource.mutateAsync({ resourceId, name: commandName, markdown: nextMarkdown });
      showSuccessToast(AGENT_RESOURCES_COPY.skillUpdated);
    } else {
      await createResource.mutateAsync({
        kind: "KIND_SKILL",
        name: commandName,
        enabled: true,
        markdown: nextMarkdown,
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
