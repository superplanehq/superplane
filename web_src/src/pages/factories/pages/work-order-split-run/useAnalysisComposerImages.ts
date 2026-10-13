import { useRef, useState, type ClipboardEvent } from "react";

import type { FilesFile } from "@/api-client";
import { isSupportedImageFile, MAX_IMAGE_ATTACHMENTS } from "@/components/AgentSidebar/useImageAttachments";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import type { HostedVideo } from "@/lib/hostedVideo";
import { showErrorToast } from "@/lib/toast";
import { revokeWorkOrderFilePreviewUrl } from "@/lib/workOrderFiles";

import {
  countCreateWorkOrderRequestImages,
  insertHostedVideoMarkdown,
  mergeCreateWorkOrderRequestImages,
  selectCreateWorkOrderRequestUploads,
} from "../../lib/createWorkOrderRequestImages";
import type { IntentAnalysisChat } from "./intentAnalysisChat";

export function addAnalysisHostedVideo(
  analysis: IntentAnalysisChat,
  pending: UploadedWorkOrderFile[],
  video: HostedVideo,
): boolean {
  const next = insertHostedVideoMarkdown(analysis.composer ?? "", video, pending);
  if (next == null) {
    showErrorToast(`Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images, videos, or audio files.`);
    return false;
  }
  if (next !== analysis.composer) {
    analysis.onComposerChange(next);
  }
  return true;
}

export function useAnalysisComposerImages({
  disabled,
  onUploadFiles,
  markdown = "",
}: {
  disabled: boolean;
  onUploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
  markdown?: string;
}) {
  const [pending, setPending] = useState<UploadedWorkOrderFile[]>([]);
  const [uploadedFiles, setUploadedFiles] = useState<UploadedWorkOrderFile[]>([]);
  const pendingRef = useRef(pending);
  pendingRef.current = pending;
  const markdownRef = useRef(markdown);
  markdownRef.current = markdown;

  const attach = async (files: FileList | File[]) => {
    if (!onUploadFiles || disabled) {
      return;
    }
    const selected = selectCreateWorkOrderRequestUploads(
      files,
      countCreateWorkOrderRequestImages(markdownRef.current, pendingRef.current),
    );
    if (selected.rejectedCount > 0) {
      showErrorToast(`Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images, videos, or audio files.`);
    }
    if (selected.accepted.length === 0) {
      return;
    }
    const uploaded = await onUploadFiles(selected.accepted);
    if (uploaded.length === 0) {
      return;
    }
    setPending((current) => [...current, ...uploaded]);
    setUploadedFiles((current) => mergeUploadedFiles(current, uploaded));
  };

  return {
    pending,
    pendingFiles: pending.filter((file) => !file.isImage && !file.isVideo && !file.isAudio),
    previewImages: mergeCreateWorkOrderRequestImages([], pending),
    transcriptFiles: uploadedFiles.map(uploadedWorkOrderFileAsTranscriptFile),
    canAttach: Boolean(onUploadFiles) && !disabled,
    attach,
    remove: (id: string) => {
      revokeWorkOrderFilePreviewUrl(id);
      setPending((current) => current.filter((file) => file.id !== id));
    },
    takePending: () => {
      const snapshot = pendingRef.current;
      setPending([]);
      pendingRef.current = [];
      return snapshot;
    },
    restorePending: (files: UploadedWorkOrderFile[]) => {
      setPending(files);
      pendingRef.current = files;
    },
    handlePaste: (event: ClipboardEvent<HTMLTextAreaElement>) => {
      const images = clipboardImageFiles(event);
      if (images.length === 0) {
        return;
      }
      if (event.clipboardData.getData("text/plain").length === 0) {
        event.preventDefault();
      }
      void attach(images);
    },
  };
}

export function mergeAnalysisTranscriptFiles(
  files: FilesFile[] | undefined,
  uploaded: FilesFile[],
): FilesFile[] | undefined {
  if (uploaded.length === 0) {
    return files;
  }
  if (!files?.length) {
    return uploaded;
  }
  const seen = new Set(files.map((file) => file.id));
  return [...files, ...uploaded.filter((file) => !seen.has(file.id))];
}

function mergeUploadedFiles(current: UploadedWorkOrderFile[], incoming: UploadedWorkOrderFile[]) {
  const seen = new Set(current.map((file) => file.id));
  return [...current, ...incoming.filter((file) => !seen.has(file.id))];
}

function uploadedWorkOrderFileAsTranscriptFile(file: UploadedWorkOrderFile): FilesFile {
  return {
    id: file.id,
    filename: file.filename,
    contentType: file.contentType,
    downloadUrl: file.previewUrl,
  };
}

function clipboardImageFiles(event: ClipboardEvent<HTMLTextAreaElement>): File[] {
  const files = event.clipboardData?.files;
  if (!files || files.length === 0) {
    return [];
  }
  return Array.from(files).filter(isSupportedImageFile);
}
