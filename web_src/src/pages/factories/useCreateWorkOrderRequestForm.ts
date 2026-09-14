import { useCallback, useEffect, useState } from "react";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

import type { CreateWorkOrderRequestDraft } from "./CreateWorkOrderRequestDialog";
import { appendUploadedWorkOrderFiles } from "./lib/createWorkOrderRequestFiles";
import { removeCreateWorkOrderRequestMarkdownImage } from "./lib/createWorkOrderRequestImages";
import { derivedWorkOrderTitle, MAX_DERIVED_WORK_ORDER_TITLE_LENGTH } from "./lib/derivedWorkOrderTitle";

export function useCreateWorkOrderRequestForm({
  open,
  description,
  isCreating,
  isUploading,
  onDescriptionChange,
  onCreate,
  onUploadFiles,
  initialAttachedFiles,
}: {
  open: boolean;
  description: string;
  isCreating: boolean;
  isUploading: boolean;
  onDescriptionChange: (next: string) => void;
  onCreate: (draft: CreateWorkOrderRequestDraft) => void;
  onUploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
  initialAttachedFiles: UploadedWorkOrderFile[];
}) {
  const [title, setTitle] = useState("");
  const [titleDirty, setTitleDirty] = useState(false);
  const [attachedFiles, setAttachedFiles] = useState<UploadedWorkOrderFile[]>(initialAttachedFiles);
  const busy = isCreating || isUploading;
  const derivedTitle = derivedWorkOrderTitle(description);
  const titleValue = titleDirty ? title : derivedTitle;
  const canCreate = Boolean(description.trim() || titleValue.trim() || attachedFiles.length) && !busy;

  const createDraft = useCallback(
    (): CreateWorkOrderRequestDraft => ({
      title: titleValue.trim() || derivedTitle,
      description: appendUploadedWorkOrderFiles(description, attachedFiles).trim(),
    }),
    [attachedFiles, derivedTitle, description, titleValue],
  );

  const submitDraft = useCallback(() => {
    if (!canCreate) {
      return;
    }
    onCreate(createDraft());
  }, [canCreate, createDraft, onCreate]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Enter" || !(event.metaKey || event.ctrlKey) || !canCreate) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      onCreate(createDraft());
    };
    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, [canCreate, createDraft, onCreate, open]);

  const handleTitleChange = (next: string) => {
    setTitleDirty(true);
    setTitle(next.replace(/[\r\n]+/g, " ").slice(0, MAX_DERIVED_WORK_ORDER_TITLE_LENGTH));
  };

  const handleAttach = async (files: FileList | File[]) => {
    if (!onUploadFiles || busy) {
      return;
    }
    const uploaded = (await onUploadFiles(files)).filter((file) => file.isImage);
    if (uploaded.length === 0) {
      return;
    }
    setAttachedFiles((current) => [...current, ...uploaded]);
  };

  const handleRemoveAttachment = (id: string) => {
    setAttachedFiles((current) => current.filter((file) => file.id !== id));
    const nextDescription = removeCreateWorkOrderRequestMarkdownImage(description, id);
    if (nextDescription !== description) {
      onDescriptionChange(nextDescription);
    }
  };

  return {
    attachedFiles,
    busy,
    canCreate,
    derivedTitle,
    titleDirty,
    titleValue,
    handleAttach,
    handleRemoveAttachment,
    handleTitleChange,
    submitDraft,
  };
}
