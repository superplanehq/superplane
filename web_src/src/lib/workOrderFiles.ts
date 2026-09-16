import type { FilesFile } from "@/api-client";

export const FILE_REF_SCHEME = "sp-file";
export const MAX_WORK_ORDER_FILE_BYTES = 50 * 1024 * 1024;
export const MAX_WORK_ORDER_FILES = 20;

export const ALLOWED_WORK_ORDER_IMAGE_TYPES = ["image/png", "image/jpeg", "image/gif", "image/webp"] as const;

export const ALLOWED_WORK_ORDER_VIDEO_TYPES = [
  "video/mp4",
  "video/webm",
  "video/quicktime",
  "video/ogg",
  "video/x-m4v",
  "video/x-matroska",
] as const;

export const ALLOWED_WORK_ORDER_FILE_TYPES = [
  ...ALLOWED_WORK_ORDER_IMAGE_TYPES,
  "application/pdf",
  "text/plain",
  "text/markdown",
  ...ALLOWED_WORK_ORDER_VIDEO_TYPES,
] as const;

const VIDEO_EXTENSIONS = [".mp4", ".webm", ".mov", ".ogv", ".ogg", ".m4v", ".mkv"] as const;

export const WORK_ORDER_FILE_ACCEPT = [
  ...ALLOWED_WORK_ORDER_FILE_TYPES,
  ".png",
  ".jpg",
  ".jpeg",
  ".gif",
  ".webp",
  ".pdf",
  ".txt",
  ".md",
  ...VIDEO_EXTENSIONS,
].join(",");

export const WORK_ORDER_VISUAL_FILE_ACCEPT = [
  ...ALLOWED_WORK_ORDER_IMAGE_TYPES,
  ...ALLOWED_WORK_ORDER_VIDEO_TYPES,
  ".png",
  ".jpg",
  ".jpeg",
  ".gif",
  ".webp",
  ...VIDEO_EXTENSIONS,
].join(",");

const previewUrls = new Map<string, string>();
const downloadUrls = new Map<string, string>();
const DOWNLOAD_URL_REFRESH_WINDOW_MS = 60_000;
const DOWNLOAD_URL_CACHE_LIMIT = 200;

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
  return Boolean(workOrderUploadContentType(file)) && file.size > 0;
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

export function isInlineWorkOrderVideo(contentType: string | undefined): boolean {
  return (ALLOWED_WORK_ORDER_VIDEO_TYPES as readonly string[]).includes(normalizeWorkOrderFileType(contentType));
}

export function isInlineWorkOrderMedia(contentType: string | undefined): boolean {
  return isInlineWorkOrderImage(contentType) || isInlineWorkOrderVideo(contentType);
}

export function looksLikeWorkOrderVideoName(name: string | undefined): boolean {
  const value = (name ?? "").split("?")[0]?.toLowerCase() ?? "";
  return VIDEO_EXTENSIONS.some((extension) => value.endsWith(extension));
}

export function isWorkOrderVideoSource(args: { contentType?: string; src?: string; alt?: string }): boolean {
  if (isInlineWorkOrderVideo(args.contentType)) {
    return true;
  }
  return looksLikeWorkOrderVideoName(args.alt) || looksLikeWorkOrderVideoName(args.src);
}

export function workOrderUploadContentType(file: File): string {
  const type = normalizeWorkOrderFileType(file.type);
  if ((ALLOWED_WORK_ORDER_FILE_TYPES as readonly string[]).includes(type)) {
    return type;
  }
  return contentTypeFromFilename(file.name);
}

export function workOrderFileContentTypeMap(files: WorkOrderFileRef[] | undefined): Record<string, string> {
  const types: Record<string, string> = {};
  for (const file of files ?? []) {
    if (file.id && file.contentType) {
      types[file.id] = file.contentType;
    }
  }
  return types;
}

export function workOrderFileContentTypeForSrc(
  src: string | undefined,
  files?: WorkOrderFileRef[],
  contentTypes?: Record<string, string>,
): string | undefined {
  const id = parseWorkOrderFileId(src);
  if (id) {
    return contentTypes?.[id] ?? files?.find((file) => file.id === id)?.contentType;
  }
  if (!src) {
    return undefined;
  }
  for (const file of files ?? []) {
    if (file.downloadUrl && (file.downloadUrl === src || src.startsWith(file.downloadUrl.split("?")[0] ?? ""))) {
      return file.contentType;
    }
  }
  for (const [fileId, type] of Object.entries(contentTypes ?? {})) {
    if (src.includes(fileId)) {
      return type;
    }
  }
  return undefined;
}

export function normalizeWorkOrderFileType(contentType: string | undefined): string {
  const value = (contentType ?? "").toLowerCase().split(";")[0]?.trim() ?? "";
  if (value === "image/jpg") {
    return "image/jpeg";
  }
  if (value === "video/x-mp4") {
    return "video/mp4";
  }
  return value;
}

function contentTypeFromFilename(filename: string): string {
  const name = filename.toLowerCase();
  if (name.endsWith(".png")) return "image/png";
  if (name.endsWith(".jpg") || name.endsWith(".jpeg")) return "image/jpeg";
  if (name.endsWith(".gif")) return "image/gif";
  if (name.endsWith(".webp")) return "image/webp";
  if (name.endsWith(".pdf")) return "application/pdf";
  if (name.endsWith(".txt")) return "text/plain";
  if (name.endsWith(".md") || name.endsWith(".markdown")) return "text/markdown";
  if (name.endsWith(".mp4")) return "video/mp4";
  if (name.endsWith(".webm")) return "video/webm";
  if (name.endsWith(".mov")) return "video/quicktime";
  if (name.endsWith(".ogv") || name.endsWith(".ogg")) return "video/ogg";
  if (name.endsWith(".m4v")) return "video/x-m4v";
  if (name.endsWith(".mkv")) return "video/x-matroska";
  return "";
}

export function setWorkOrderFilePreviewUrl(id: string, url: string): void {
  previewUrls.set(id, url);
}

export function clearWorkOrderFileDownloadCache(): void {
  downloadUrls.clear();
}

export function workOrderFileDownloadUrlIsFresh(url: string): boolean {
  const expiresAt = signedDownloadUrlExpiresAt(url);
  if (expiresAt === undefined) {
    return true;
  }
  return expiresAt - Date.now() >= DOWNLOAD_URL_REFRESH_WINDOW_MS;
}

export function workOrderFileDownloadMap(files: WorkOrderFileRef[] | undefined): Record<string, string> {
  const urls: Record<string, string> = {};
  for (const file of files ?? []) {
    if (file.id && file.downloadUrl) {
      urls[file.id] = stableWorkOrderFileDownloadUrl(file.id, file.downloadUrl);
    }
  }
  return urls;
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
  const fileDownloadUrls = workOrderFileDownloadMap(files);
  let next = markdown;
  for (const [id, downloadUrl] of Object.entries(fileDownloadUrls)) {
    next = next.split(workOrderFileRef(id)).join(downloadUrl);
  }
  return next;
}

function stableWorkOrderFileDownloadUrl(id: string, nextUrl: string): string {
  const currentUrl = downloadUrls.get(id);
  if (!currentUrl || currentUrl === nextUrl || downloadUrlResource(currentUrl) !== downloadUrlResource(nextUrl)) {
    return rememberWorkOrderFileDownloadUrl(id, nextUrl);
  }

  const expiresAt = signedDownloadUrlExpiresAt(currentUrl);
  if (expiresAt !== undefined && expiresAt - Date.now() < DOWNLOAD_URL_REFRESH_WINDOW_MS) {
    return rememberWorkOrderFileDownloadUrl(id, nextUrl);
  }

  return rememberWorkOrderFileDownloadUrl(id, currentUrl);
}

function rememberWorkOrderFileDownloadUrl(id: string, url: string): string {
  downloadUrls.delete(id);
  downloadUrls.set(id, url);
  if (downloadUrls.size > DOWNLOAD_URL_CACHE_LIMIT) {
    const oldestId = downloadUrls.keys().next().value;
    if (oldestId !== undefined) {
      downloadUrls.delete(oldestId);
    }
  }
  return url;
}

function downloadUrlResource(rawUrl: string): string {
  try {
    const url = new URL(rawUrl, "http://localhost");
    return `${url.origin}${url.pathname}`;
  } catch {
    return rawUrl.split("?")[0] ?? rawUrl;
  }
}

function signedDownloadUrlExpiresAt(rawUrl: string): number | undefined {
  try {
    const url = new URL(rawUrl, "http://localhost");
    const superPlaneExpires = Number(url.searchParams.get("expires"));
    if (Number.isFinite(superPlaneExpires) && superPlaneExpires > 0) {
      return superPlaneExpires * 1_000;
    }

    const signedAt = parseGcsSigningTime(url.searchParams.get("X-Goog-Date"));
    const lifetimeSeconds = Number(url.searchParams.get("X-Goog-Expires"));
    if (signedAt !== undefined && Number.isFinite(lifetimeSeconds) && lifetimeSeconds >= 0) {
      return signedAt + lifetimeSeconds * 1_000;
    }
  } catch {
    return undefined;
  }
  return undefined;
}

function parseGcsSigningTime(value: string | null): number | undefined {
  const match = /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/.exec(value ?? "");
  if (!match) {
    return undefined;
  }
  const [, year, month, day, hour, minute, second] = match;
  const timestamp = Date.UTC(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second),
  );
  return Number.isNaN(timestamp) ? undefined : timestamp;
}
