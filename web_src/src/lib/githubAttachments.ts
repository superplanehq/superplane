import { isGitHubWebHostname } from "./githubHost";

export function isGitHubUserAttachmentUrl(raw: string | undefined, host?: string | null): boolean {
  const value = raw?.trim() ?? "";
  if (!value) {
    return false;
  }

  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    return false;
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    return false;
  }
  if (!isGitHubWebHostname(parsed.hostname, host)) {
    return false;
  }

  return isGitHubAttachmentPath(parsed.pathname);
}

function isGitHubAttachmentPath(pathname: string): boolean {
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length < 3) {
    return false;
  }

  const root = segments[0].toLowerCase();
  const second = segments[1].toLowerCase();
  if (root === "user-attachments") {
    return second === "assets" || second === "files";
  }

  return segments.length >= 4 && segments[2].toLowerCase() === "assets";
}

export function isGitHubAttachmentAutolink(href: string | undefined, label: string, host?: string | null): boolean {
  if (!isGitHubUserAttachmentUrl(href, host) || !href) {
    return false;
  }

  const text = label.trim();
  if (!text) {
    return false;
  }

  const target = href.trim();
  return text === target || text === target.replace(/^https?:\/\//, "");
}
