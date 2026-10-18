import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
  useUpdateFactoryAgentResource,
  useRestoreFactorySystemSkill,
} from "@/hooks/useFactoryAgentResources";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryAgentPath, factoryRouteSegment } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import {
  sanitizeSkillCommandName,
  setSkillFrontmatterFields,
  skillDisplayTitle,
  skillFrontmatterField,
} from "./skillFrontmatter";

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,62}$/;
const RESERVED_NAME = "superplane";
const SETUP_NAME = "setup";
const EMPTY_SKILL_MARKDOWN = "---\nname: \ntitle: \ndescription: \n---\n\n";

export { sanitizeSkillCommandName };

export function skillCommandForSave(title: string, storedName: string, nameTouched: boolean): string {
  if (!nameTouched && storedName.trim()) {
    return storedName.trim();
  }
  return sanitizeSkillCommandName(title);
}

export function validateSkillName(name: string): string {
  if (!name) {
    return AGENT_RESOURCES_COPY.nameRequired;
  }
  if (name === RESERVED_NAME || name === SETUP_NAME) {
    return AGENT_RESOURCES_COPY.nameReserved(name);
  }
  if (!NAME_PATTERN.test(name)) {
    return AGENT_RESOURCES_COPY.nameInvalid;
  }
  return "";
}

export function useSkillEditorPage() {
  const { resourceId } = useParams<{ resourceId?: string }>();
  const isCreate = !resourceId || resourceId === "new";
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const navigate = useNavigate();
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const resource = isCreate ? undefined : skills.data?.find((entry) => entry.id === resourceId);
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const [nameError, setNameError] = useState("");
  const [markdownError, setMarkdownError] = useState("");
  const [pendingDelete, setPendingDelete] = useState(false);
  const listPath = factoryAgentPath(organizationId, factoryRouteSegment(factory));
  const draft = useSkillEditorDraft(isCreate, resource);
  const restoration = useSkillRestore(organizationId, factoryId, resource, (restored) => {
    draft.replaceDraft(restored);
    setNameError("");
    setMarkdownError("");
  });

  return {
    isCreate,
    factory,
    canUpdate,
    resource,
    skillsLoading: skills.isLoading,
    name: draft.name,
    markdown: draft.markdown,
    nameError,
    markdownError,
    pendingDelete,
    ...restoration,
    listPath,
    commandName: draft.commandName,
    command: draft.command,
    isSaving: createResource.isPending || updateResource.isPending,
    isDeleting: deleteResource.isPending,
    setName: draft.setName,
    setMarkdown: draft.setMarkdown,
    setPendingDelete,
    navigateToList: () => navigate(listPath),
    handleSave: () =>
      saveSkill({
        commandName: draft.commandName,
        title: draft.name.trim(),
        markdown: draft.markdown,
        taken: Boolean(
          draft.commandName &&
            skills.data?.some((entry) => entry.id !== resource?.id && entry.name === draft.commandName),
        ),
        resourceId: resource?.id,
        listPath,
        navigate,
        setNameError,
        setMarkdownError,
        createResource,
        updateResource,
      }),
    confirmDelete: () => deleteSkill({ resourceId: resource?.id, listPath, navigate, deleteResource }),
    toggleEnabled: (enabled: boolean) => {
      if (!resource?.id) {
        return;
      }
      void updateResource.mutateAsync({ resourceId: resource.id, enabled }).catch((error) => {
        showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.skillUpdateFailed));
      });
    },
  };
}

function useSkillEditorDraft(isCreate: boolean, resource: FactoriesFactoryAgentResource | undefined) {
  const [name, setName] = useState("");
  const [markdown, setMarkdown] = useState("");
  const [storedCommand, setStoredCommand] = useState("");
  const [nameTouched, setNameTouched] = useState(false);
  const [draftKey, setDraftKey] = useState("");
  const markdownRef = useRef("");
  const frontmatterRef = useRef<SkillNameFields>({ name: "", title: "" });
  const writeSkillMarkdown = useCallback((next: string | ((current: string) => string)) => {
    const value = typeof next === "function" ? next(markdownRef.current) : next;
    markdownRef.current = value;
    frontmatterRef.current = readSkillNameFields(value);
    setMarkdown(value);
  }, []);
  const setMarkdownFromEditor = useCallback(
    (value: string) => {
      const previous = frontmatterRef.current;
      writeSkillMarkdown(value);
      adoptChangedSkillName(previous, frontmatterRef.current, (nextName, command) => {
        setName(nextName);
        setStoredCommand(command);
        setNameTouched(false);
      });
    },
    [writeSkillMarkdown],
  );
  const replaceDraft = useCallback(
    (next: FactoriesFactoryAgentResource) => {
      setName(skillDisplayTitle(next));
      setStoredCommand(next.name ?? "");
      setNameTouched(false);
      writeSkillMarkdown(next.markdown ?? "");
    },
    [writeSkillMarkdown],
  );

  useEffect(() => {
    syncLoadedSkillDraft(isCreate, resource, draftKey, {
      setName,
      setStoredCommand,
      setNameTouched,
      setDraftKey,
      writeSkillMarkdown,
    });
  }, [isCreate, resource, draftKey, writeSkillMarkdown]);

  const commandName = skillCommandForSave(name, storedCommand, nameTouched);
  return {
    name,
    markdown,
    commandName,
    command: commandName ? `/${commandName}` : "/",
    setName: (value: string) => {
      setNameTouched(true);
      setNameAndFrontmatter(value, setName, writeSkillMarkdown);
    },
    setMarkdown: setMarkdownFromEditor,
    replaceDraft,
  };
}

function useSkillRestore(
  organizationId: string,
  factoryId: string,
  resource: FactoriesFactoryAgentResource | undefined,
  onRestored: (resource: FactoriesFactoryAgentResource) => void,
) {
  const restoreResource = useRestoreFactorySystemSkill(organizationId, factoryId);
  const activeResourceId = useRef(resource?.id);
  activeResourceId.current = resource?.id;
  const [restoreTargetId, setRestoreTargetId] = useState<string>();
  return {
    pendingRestore: Boolean(restoreTargetId && restoreTargetId === resource?.id),
    setPendingRestore: (open: boolean) => setRestoreTargetId(open ? resource?.id : undefined),
    isRestoring: restoreResource.isPending,
    canRestore:
      resource?.systemSkill?.state === "STATE_CUSTOM" || resource?.systemSkill?.state === "STATE_UPDATE_AVAILABLE",
    confirmRestore: async () => {
      if (!resource?.id) return;
      try {
        const restored = await restoreResource.mutateAsync(resource.id);
        setRestoreTargetId(undefined);
        if (activeResourceId.current !== restored.id) return;
        onRestored(restored);
        showSuccessToast(AGENT_RESOURCES_COPY.restoreSkillSuccess);
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.restoreSkillFailed));
      }
    },
  };
}

function syncLoadedSkillDraft(
  isCreate: boolean,
  resource: FactoriesFactoryAgentResource | undefined,
  draftKey: string,
  writers: SkillDraftWriters,
) {
  const nextDraftKey = isCreate ? "new" : (resource?.id ?? "");
  if ((!isCreate && !resource) || !nextDraftKey || draftKey === nextDraftKey) {
    return;
  }
  if (isCreate) {
    writers.setName("");
    writers.setStoredCommand("");
    writers.setNameTouched(false);
    writers.writeSkillMarkdown(EMPTY_SKILL_MARKDOWN);
    writers.setDraftKey("new");
    return;
  }
  if (!resource) {
    return;
  }
  writers.setName(skillDisplayTitle(resource));
  writers.setStoredCommand((resource.name ?? "").trim());
  writers.setNameTouched(false);
  writers.writeSkillMarkdown(resource.markdown ?? "");
  writers.setDraftKey(nextDraftKey);
}

function adoptChangedSkillName(
  previous: SkillNameFields,
  fields: SkillNameFields,
  apply: (name: string, command: string) => void,
) {
  if (!skillNameFieldsChanged(previous, fields)) {
    return;
  }
  const adopted = skillNameFromFile(fields);
  if (!adopted) {
    return;
  }
  apply(adopted.name, adopted.command);
}

type SkillDraftWriters = {
  setName: (value: string) => void;
  setStoredCommand: (value: string) => void;
  setNameTouched: (value: boolean) => void;
  setDraftKey: (value: string) => void;
  writeSkillMarkdown: (value: string) => void;
};

type SkillNameFields = {
  name: string;
  title: string;
};

function readSkillNameFields(markdown: string): SkillNameFields {
  return {
    name: skillFrontmatterField(markdown, "name")?.trim() ?? "",
    title: skillFrontmatterField(markdown, "title")?.trim() ?? "",
  };
}

function skillNameFieldsChanged(previous: SkillNameFields, next: SkillNameFields): boolean {
  return previous.name !== next.name || previous.title !== next.title;
}

function skillNameFromFile(fields: SkillNameFields): { name: string; command: string } | undefined {
  if (!fields.name && !fields.title) {
    return undefined;
  }
  const name = fields.title || fields.name;
  const command = fields.name ? sanitizeSkillCommandName(fields.name) : sanitizeSkillCommandName(name);
  return { name, command };
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
