import { useCallback, useEffect, useRef, useState } from "react";

import { MAX_IMAGE_ATTACHMENTS } from "@/components/AgentSidebar/useImageAttachments";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { showErrorToast } from "@/lib/toast";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import type { CreateWorkOrderRequestDraft } from "./CreateWorkOrderRequestDialog";
import {
  appendUploadedWorkOrderImages,
  countCreateWorkOrderRequestImages,
  removeCreateWorkOrderRequestMarkdownImage,
  selectCreateWorkOrderRequestUploads,
} from "./lib/createWorkOrderRequestImages";
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
  const attachedFilesRef = useRef(attachedFiles);
  attachedFilesRef.current = attachedFiles;
  const busy = isCreating || isUploading;
  const derivedTitle = derivedWorkOrderTitle(description);
  const titleValue = titleDirty ? title : derivedTitle;
  const canCreate = Boolean(description.trim() || titleValue.trim() || attachedFiles.length) && !busy;

  const createDraft = useCallback(
    (): CreateWorkOrderRequestDraft => ({
      title: titleValue.trim() || derivedTitle || CREATE_WORK_ORDER_REQUEST_COPY.title,
      description: appendUploadedWorkOrderImages(description, attachedFiles).trim(),
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

  const uploadAcceptedFiles = async (files: FileList | File[]): Promise<UploadedWorkOrderFile[]> => {
    if (!onUploadFiles || busy) {
      return [];
    }
    const selected = selectCreateWorkOrderRequestUploads(
      files,
      countCreateWorkOrderRequestImages(description, attachedFilesRef.current),
    );
    if (selected.rejectedCount > 0) {
      showErrorToast(`Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images.`);
    }
    if (selected.accepted.length === 0) {
      return [];
    }
    return onUploadFiles(selected.accepted);
  };

  const handleAttach = async (files: FileList | File[]) => {
    const uploaded = (await uploadAcceptedFiles(files)).filter((file) => file.isImage || file.isVideo);
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
    canAttach: !busy && countCreateWorkOrderRequestImages(description, attachedFiles) < MAX_IMAGE_ATTACHMENTS,
    canCreate,
    derivedTitle,
    titleDirty,
    titleValue,
    handleAttach,
    handleRemoveAttachment,
    handleTitleChange,
    submitDraft,
    uploadAcceptedFiles,
  };
}
