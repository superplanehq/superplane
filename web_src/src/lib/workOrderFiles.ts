import type { FilesFile } from "@/api-client";

export const FILE_REF_SCHEME = "sp-file";
export const MAX_WORK_ORDER_FILE_BYTES = 10 * 1024 * 1024;
export const MAX_WORK_ORDER_FILES = 20;

export const ALLOWED_WORK_ORDER_FILE_TYPES = [
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "image/heic",
  "image/heif",
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "text/plain",
  "text/markdown",
  "application/json",
  "text/csv",
  "application/yaml",
] as const;

const WORK_ORDER_FILE_TYPES_BY_EXTENSION: Record<string, string> = {
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".heic": "image/heic",
  ".heif": "image/heif",
  ".pdf": "application/pdf",
  ".doc": "application/msword",
  ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  ".txt": "text/plain",
  ".md": "text/markdown",
  ".json": "application/json",
  ".csv": "text/csv",
  ".yaml": "application/yaml",
  ".yml": "application/yaml",
};

const WORK_ORDER_FILE_EXTENSIONS = Object.keys(WORK_ORDER_FILE_TYPES_BY_EXTENSION);

/** Accept attribute for work-order file inputs: every allowed type plus the extensions some browsers map to them. */
export const WORK_ORDER_FILE_ACCEPT = [...ALLOWED_WORK_ORDER_FILE_TYPES, ...WORK_ORDER_FILE_EXTENSIONS].join(",");

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

function isAllowedWorkOrderContentType(contentType: string): boolean {
  return (ALLOWED_WORK_ORDER_FILE_TYPES as readonly string[]).includes(normalizeWorkOrderFileType(contentType));
}

export function isAllowedWorkOrderFile(file: File): boolean {
  return isAllowedWorkOrderContentType(resolveWorkOrderFileMimeType(file)) && file.size > 0;
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
  if (value === "text/yaml" || value === "application/x-yaml") {
    return "application/yaml";
  }
  if (value === "text/x-csv" || value === "application/csv" || value === "text/comma-separated-values") {
    return "text/csv";
  }
  return value;
}

/** Resolves the MIME type a file is stored with. Browsers send an empty type or application/octet-stream for many text files, so the filename decides. */
export function resolveWorkOrderFileMimeType(file: Pick<File, "name" | "type">): string {
  const declared = normalizeWorkOrderFileType(file.type);
  if (file.name.toLowerCase().endsWith(".csv") && declared === "application/vnd.ms-excel") {
    return "text/csv";
  }
  if (declared && declared !== "application/octet-stream") {
    return declared;
  }
  return workOrderMimeTypeFromFilename(file.name);
}

export function workOrderMimeTypeFromFilename(filename: string): string {
  const extension = filename.slice(filename.lastIndexOf(".")).toLowerCase();
  if (!extension.startsWith(".")) {
    return "";
  }
  return WORK_ORDER_FILE_TYPES_BY_EXTENSION[extension] ?? "";
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
