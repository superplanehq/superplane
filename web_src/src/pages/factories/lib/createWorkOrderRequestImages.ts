import { isSupportedImageFile, MAX_IMAGE_ATTACHMENTS } from "@/components/AgentSidebar/useImageAttachments";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import {
  isInlineWorkOrderVideo,
  isWorkOrderVideoSource,
  parseWorkOrderFileId,
  resolveWorkOrderFileSrc,
  workOrderUploadContentType,
} from "@/lib/workOrderFiles";

const MARKDOWN_IMAGE = /!\[([^\]]*)\]\(([^)]+)\)/g;

export interface CreateWorkOrderRequestImage {
  id: string;
  alt: string;
  src: string;
  isVideo?: boolean;
}

export function createWorkOrderRequestImages(
  markdown: string,
  fileUrls?: Record<string, string>,
): CreateWorkOrderRequestImage[] {
  const images: CreateWorkOrderRequestImage[] = [];
  const seen = new Set<string>();
  for (const match of markdown.matchAll(MARKDOWN_IMAGE)) {
    const alt = match[1] ?? "";
    const rawSrc = match[2]?.trim() ?? "";
    if (!rawSrc || seen.has(rawSrc)) {
      continue;
    }
    const src = resolveWorkOrderFileSrc(rawSrc, fileUrls);
    if (!src) {
      continue;
    }
    seen.add(rawSrc);
    images.push({
      id: parseWorkOrderFileId(rawSrc) ?? rawSrc,
      alt,
      src,
      isVideo: isWorkOrderVideoSource({ src, alt }),
    });
  }
  return images;
}

export function countCreateWorkOrderRequestImages(markdown: string, attached: UploadedWorkOrderFile[] = []): number {
  const seen = new Set<string>();
  for (const match of markdown.matchAll(MARKDOWN_IMAGE)) {
    const rawSrc = match[2]?.trim() ?? "";
    if (!rawSrc) {
      continue;
    }
    seen.add(parseWorkOrderFileId(rawSrc) ?? rawSrc);
  }
  for (const file of attached) {
    if (file.isImage || file.isVideo) {
      seen.add(file.id);
    }
  }
  return seen.size;
}

export function selectCreateWorkOrderRequestUploads(
  files: FileList | File[],
  currentImageCount: number,
): { accepted: File[]; rejectedCount: number } {
  const remaining = Math.max(0, MAX_IMAGE_ATTACHMENTS - currentImageCount);
  const visual: File[] = [];
  const others: File[] = [];
  for (const file of Array.from(files)) {
    if (isSupportedImageFile(file) || isInlineWorkOrderVideo(workOrderUploadContentType(file))) {
      visual.push(file);
    } else {
      others.push(file);
    }
  }
  const acceptedVisual = visual.slice(0, remaining);
  return {
    accepted: [...acceptedVisual, ...others],
    rejectedCount: visual.length - acceptedVisual.length,
  };
}

export function mergeCreateWorkOrderRequestImages(
  fromDescription: CreateWorkOrderRequestImage[],
  attached: UploadedWorkOrderFile[],
): CreateWorkOrderRequestImage[] {
  const images = [...fromDescription];
  const seen = new Set(fromDescription.map((image) => image.id));
  for (const file of attached) {
    if ((!file.isImage && !file.isVideo) || seen.has(file.id)) {
      continue;
    }
    const src = file.previewUrl || resolveWorkOrderFileSrc(file.ref);
    if (!src) {
      continue;
    }
    seen.add(file.id);
    images.push({
      id: file.id,
      alt: file.filename,
      src,
      isVideo:
        Boolean(file.isVideo) || isWorkOrderVideoSource({ contentType: file.contentType, src, alt: file.filename }),
    });
  }
  return images;
}

export function appendUploadedWorkOrderImages(description: string, files: UploadedWorkOrderFile[]): string {
  const blocks = files.map((file) =>
    file.isImage || file.isVideo
      ? `![${markdownFileLabel(file.filename)}](${file.ref})`
      : `[${markdownFileLabel(file.filename)}](${file.ref})`,
  );
  return [description.trimEnd(), ...blocks].filter((part) => part.length > 0).join("\n\n");
}

export function removeCreateWorkOrderRequestMarkdownImage(markdown: string, id: string): string {
  const next = markdown.replace(MARKDOWN_IMAGE, (full, _alt: string, rawSrc: string) => {
    const src = rawSrc.trim();
    const imageId = parseWorkOrderFileId(src) ?? src;
    return imageId === id ? "" : full;
  });
  return next.replace(/\n{3,}/g, "\n\n").trim();
}

function markdownFileLabel(filename: string): string {
  return filename.replace(/[[\]()]/g, "").trim() || "file";
}
