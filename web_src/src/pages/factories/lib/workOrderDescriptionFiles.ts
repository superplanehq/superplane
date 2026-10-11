import type { Editor } from "@tiptap/core";

import { MAX_IMAGE_ATTACHMENTS } from "@/components/AgentSidebar/useImageAttachments";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import type { HostedVideo } from "@/lib/hostedVideo";
import { showErrorToast } from "@/lib/toast";

import { countCreateWorkOrderRequestImages } from "./createWorkOrderRequestImages";

export function insertHostedVideo(editor: Editor, video: HostedVideo): boolean {
  const markdown = editor.getMarkdown();
  if (!markdown.includes(video.pageUrl) && countCreateWorkOrderRequestImages(markdown) >= MAX_IMAGE_ATTACHMENTS) {
    showErrorToast(`Attachments are limited to ${MAX_IMAGE_ATTACHMENTS} images, videos, or audio files.`);
    return false;
  }
  if (!markdown.includes(video.pageUrl)) {
    editor.chain().focus().setImage({ src: video.pageUrl, alt: video.providerName }).run();
  }
  return true;
}

export function insertUploadedFiles(editor: Editor, files: UploadedWorkOrderFile[]): void {
  const contentTypes = { ...(editor.storage.image?.contentTypes ?? {}) };
  for (const file of files) {
    contentTypes[file.id] = file.contentType;
  }
  editor.storage.image = {
    ...(editor.storage.image ?? { downloadUrls: {} }),
    contentTypes,
  };
  for (const file of files) {
    if (file.isImage || file.isVideo || file.isAudio) {
      editor.chain().focus().setImage({ src: file.ref, alt: file.filename }).run();
    } else {
      editor.chain().focus().insertContent(`[${file.filename}](${file.ref})`).run();
    }
  }
}
