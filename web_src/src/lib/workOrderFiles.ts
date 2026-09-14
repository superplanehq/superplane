import type { FilesFile } from "@/api-client";

export const FILE_REF_SCHEME = "sp-file";
export const MAX_WORK_ORDER_FILE_BYTES = 10 * 1024 * 1024;
export const MAX_WORK_ORDER_FILES = 20;

export const ALLOWED_WORK_ORDER_FILE_TYPES = [
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "application/pdf",
  "text/plain",
  "text/markdown",
] as const;

const previewUrls = new Map<string, string>();
const stableDownloadUrls = new Map<string, string>();
const STABLE_DOWNLOAD_MIN_REMAINING_SECONDS = 60;

export type WorkOrderFileRef = Pick<FilesFile, "id" | "downloadUrl" | "filename" | "contentType">;

export function parseWorkOrderFileId(raw: string | undefined): string | null {
  const value = raw?.trim() ?? "";
  const prefix = `${FILE_REF_SCHEME}://`;
  if (!value.startsWith(prefix)) {
    return null;
  }
  const id = value.slice(prefix.length).trim();
  return id === "" ? null : id;
}

export function workOrderFileRef(id: string): string {
  return `${FILE_REF_SCHEME}://${id}`;
}

export function isAllowedWorkOrderFile(file: File): boolean {
  const type = normalizeWorkOrderFileType(file.type);
  return (ALLOWED_WORK_ORDER_FILE_TYPES as readonly string[]).includes(type) && file.size > 0;
}

export function isInlineWorkOrderImage(contentType: string | undefined): boolean {
  switch (normalizeWorkOrderFileType(contentType)) {
    case "image/png":
    case "image/jpeg":
    case "image/gif":
    case "image/webp":
      return true;
    default:
      return false;
  }
}

export function normalizeWorkOrderFileType(contentType: string | undefined): string {
  const value = (contentType ?? "").toLowerCase().split(";")[0]?.trim() ?? "";
  if (value === "image/jpg") {
    return "image/jpeg";
  }
  return value;
}

export function setWorkOrderFilePreviewUrl(id: string, url: string): void {
  previewUrls.set(id, url);
}

export function clearWorkOrderFileDownloadCache(): void {
  stableDownloadUrls.clear();
}

export function workOrderFileDownloadMap(files: WorkOrderFileRef[] | undefined): Record<string, string> {
  const urls: Record<string, string> = {};
  for (const file of files ?? []) {
    if (file.id && file.downloadUrl) {
      urls[file.id] = stableWorkOrderDownloadUrl(file.id, file.downloadUrl);
    }
  }
  return urls;
}

function stableWorkOrderDownloadUrl(id: string, nextUrl: string): string {
  const current = stableDownloadUrls.get(id);
  if (!current || current === nextUrl) {
    stableDownloadUrls.set(id, nextUrl);
    return nextUrl;
  }
  if (fileResourceKey(current) !== fileResourceKey(nextUrl)) {
    stableDownloadUrls.set(id, nextUrl);
    return nextUrl;
  }
  const remaining = signedUrlRemainingSeconds(current);
  if (remaining != null && remaining < STABLE_DOWNLOAD_MIN_REMAINING_SECONDS) {
    stableDownloadUrls.set(id, nextUrl);
    return nextUrl;
  }
  return current;
}

function fileResourceKey(url: string): string {
  try {
    const parsed = new URL(url);
    return `${parsed.origin}${parsed.pathname}`;
  } catch {
    return url;
  }
}

function signedUrlRemainingSeconds(url: string): number | null {
  try {
    const parsed = new URL(url);
    const expires = parsed.searchParams.get("expires");
    if (expires) {
      const unix = Number(expires);
      return Number.isFinite(unix) ? unix - Date.now() / 1000 : null;
    }
    const googDate = parsed.searchParams.get("X-Goog-Date");
    const googExpires = parsed.searchParams.get("X-Goog-Expires");
    if (!googDate || !googExpires) {
      return null;
    }
    const startedAt = parseGoogleSignedUrlDate(googDate);
    const ttlSeconds = Number(googExpires);
    if (startedAt == null || !Number.isFinite(ttlSeconds)) {
      return null;
    }
    return (startedAt + ttlSeconds * 1000 - Date.now()) / 1000;
  } catch {
    return null;
  }
}

function parseGoogleSignedUrlDate(value: string): number | null {
  const match = /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/.exec(value);
  if (!match) {
    return null;
  }
  const ms = Date.parse(`${match[1]}-${match[2]}-${match[3]}T${match[4]}:${match[5]}:${match[6]}Z`);
  return Number.isNaN(ms) ? null : ms;
}

export function resolveWorkOrderFileSrc(src: string | undefined, downloadUrls?: Record<string, string>): string {
  if (!src) {
    return "";
  }
  const id = parseWorkOrderFileId(src);
  if (!id) {
    return src;
  }
  return previewUrls.get(id) ?? downloadUrls?.[id] ?? src;
}

export function isReachableWorkOrderFileUrl(src: string | undefined): boolean {
  const value = src?.trim() ?? "";
  return /^(https?:|blob:)/i.test(value);
}

export function rewriteWorkOrderFileRefs(markdown: string, files: WorkOrderFileRef[] | undefined): string {
  if (!markdown || !files?.length) {
    return markdown;
  }
  const urls = workOrderFileDownloadMap(files);
  let next = markdown;
  for (const [id, downloadUrl] of Object.entries(urls)) {
    next = next.split(workOrderFileRef(id)).join(downloadUrl);
  }
  return next;
}
