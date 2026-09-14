import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { parseWorkOrderFileId, resolveWorkOrderFileSrc } from "@/lib/workOrderFiles";

const MARKDOWN_IMAGE = /!\[([^\]]*)\]\(([^)]+)\)/g;

export interface CreateWorkOrderRequestImage {
  id: string;
  alt: string;
  src: string;
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
    });
  }
  return images;
}

export function mergeCreateWorkOrderRequestImages(
  fromDescription: CreateWorkOrderRequestImage[],
  attached: UploadedWorkOrderFile[],
): CreateWorkOrderRequestImage[] {
  const images = [...fromDescription];
  const seen = new Set(fromDescription.map((image) => image.id));
  for (const file of attached) {
    if (!file.isImage || seen.has(file.id)) {
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
    });
  }
  return images;
}

export function appendUploadedWorkOrderImages(description: string, files: UploadedWorkOrderFile[]): string {
  const blocks = files
    .filter((file) => file.isImage)
    .map((file) => `![${markdownImageLabel(file.filename)}](${file.ref})`);
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

function markdownImageLabel(filename: string): string {
  return filename.replace(/[[\]()]/g, "").trim() || "image";
}
