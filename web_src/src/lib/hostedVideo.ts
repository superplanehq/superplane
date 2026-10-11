import catalog from "@runner/hosted_video_hosts.json";

export const HOSTED_VIDEO_MAX_DURATION_SECONDS = catalog.maxDurationSeconds;

export const HOSTED_VIDEO_COPY = {
  add: "Add video link",
  label: "Video link",
  placeholder: "https://",
  helper: "The video must be public and at most 5 minutes.",
  unsupported: "Use an HTTPS YouTube, Vimeo, Loom, or CleanShot link.",
  apply: "Add",
  open: "Open video",
} as const;

export interface HostedVideo {
  providerId: string;
  providerName: string;
  id: string;
  pageUrl: string;
  embedUrl: string | null;
}

const YOUTUBE_ID = /^[A-Za-z0-9_-]{11}$/;
const VIMEO_ID = /^[0-9]{6,12}$/;
const LOOM_ID = /^[A-Za-z0-9]{16,64}$/;

export function parseHostedVideoUrl(raw: string): HostedVideo | null {
  const trimmed = raw.trim();
  if (!trimmed || /\s/.test(trimmed)) {
    return null;
  }
  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    return null;
  }
  if (parsed.protocol !== "https:" || parsed.username || parsed.password) {
    return null;
  }
  const host = hostFor(parsed.hostname);
  if (!host) {
    return null;
  }
  const id = videoId(host.id, parsed);
  if (!id) {
    return null;
  }
  return {
    providerId: host.id,
    providerName: host.name,
    id,
    pageUrl: parsed.toString(),
    embedUrl: embedUrl(host.id, id),
  };
}

export function hostedVideoFromClipboard(text: string): HostedVideo | null {
  const trimmed = text.trim();
  if (!trimmed || /\s/.test(trimmed)) {
    return null;
  }
  return parseHostedVideoUrl(trimmed);
}

export function hostedVideoMarkdown(video: HostedVideo): string {
  return `![${video.providerName}](${video.pageUrl})`;
}

export function appendHostedVideoMarkdown(markdown: string, video: HostedVideo): string {
  if (markdown.includes(video.pageUrl)) {
    return markdown;
  }
  const block = hostedVideoMarkdown(video);
  const trimmed = markdown.trimEnd();
  return trimmed ? `${trimmed}\n\n${block}` : block;
}

export function hostedVideoFieldError(value: string): string {
  const trimmed = value.trim();
  if (!trimmed || !looksLikeUrl(trimmed)) {
    return "";
  }
  return parseHostedVideoUrl(trimmed) ? "" : HOSTED_VIDEO_COPY.unsupported;
}

function looksLikeUrl(value: string): boolean {
  return value.includes("://") || value.includes(".");
}

function hostFor(hostname: string): { id: string; name: string } | null {
  const host = hostname.toLowerCase().replace(/\.$/, "");
  for (const candidate of catalog.hosts) {
    if (candidate.suffixes.some((suffix) => host === suffix || host.endsWith(`.${suffix}`))) {
      return candidate;
    }
  }
  return null;
}

function videoId(providerId: string, parsed: URL): string | null {
  if (providerId === "youtube") {
    return youtubeId(parsed);
  }
  if (providerId === "vimeo") {
    return vimeoId(parsed);
  }
  if (providerId === "loom") {
    return loomId(parsed);
  }
  const parts = pathSegments(parsed.pathname);
  const id = parts[parts.length - 1] ?? "";
  return id.length >= 4 ? id : null;
}

function youtubeId(parsed: URL): string | null {
  const host = parsed.hostname.toLowerCase();
  if (host === "youtu.be" || host.endsWith(".youtu.be")) {
    const id = pathSegments(parsed.pathname)[0] ?? "";
    return YOUTUBE_ID.test(id) ? id : null;
  }
  const parts = pathSegments(parsed.pathname);
  if (parts.length === 1 && parts[0] === "watch") {
    const id = parsed.searchParams.get("v") ?? "";
    return YOUTUBE_ID.test(id) ? id : null;
  }
  if (
    parts.length >= 2 &&
    ["shorts", "embed", "live", "v"].includes(parts[0] ?? "") &&
    YOUTUBE_ID.test(parts[1] ?? "")
  ) {
    return parts[1] ?? null;
  }
  return null;
}

function vimeoId(parsed: URL): string | null {
  const parts = pathSegments(parsed.pathname);
  return (
    labeledVimeoId(parts, "video") ||
    trailingVimeoId(parts, "videos") ||
    trailingVimeoId(parts, "video") ||
    vimeoIdAt(parts, 0)
  );
}

function labeledVimeoId(parts: string[], label: string): string | null {
  if (parts[0] !== label) {
    return null;
  }
  return vimeoIdAt(parts, 1);
}

function trailingVimeoId(parts: string[], label: string): string | null {
  if (parts.length < 3 || parts[parts.length - 2] !== label) {
    return null;
  }
  return vimeoIdAt(parts, parts.length - 1);
}

function vimeoIdAt(parts: string[], index: number): string | null {
  const id = parts[index] || "";
  if (!VIMEO_ID.test(id)) {
    return null;
  }
  return id;
}

function loomId(parsed: URL): string | null {
  const parts = pathSegments(parsed.pathname);
  if (parts.length >= 2 && (parts[0] === "share" || parts[0] === "embed") && LOOM_ID.test(parts[1] ?? "")) {
    return parts[1] ?? null;
  }
  return null;
}

function embedUrl(providerId: string, id: string): string | null {
  if (providerId === "youtube") {
    return `https://www.youtube-nocookie.com/embed/${id}`;
  }
  if (providerId === "vimeo") {
    return `https://player.vimeo.com/video/${id}`;
  }
  if (providerId === "loom") {
    return `https://www.loom.com/embed/${id}`;
  }
  return null;
}

function pathSegments(pathname: string): string[] {
  return pathname
    .split("/")
    .map((part) => part.trim())
    .filter(Boolean);
}
