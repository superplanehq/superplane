export function isGitHubUserAttachmentUrl(raw: string | undefined): boolean {
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
  if (parsed.hostname.toLowerCase() !== "github.com") {
    return false;
  }

  const path = parsed.pathname.toLowerCase();
  return path.includes("/user-attachments/") || path.includes("/assets/");
}

export function isGitHubAttachmentAutolink(href: string | undefined, label: string): boolean {
  if (!isGitHubUserAttachmentUrl(href) || !href) {
    return false;
  }

  const text = label.trim();
  if (!text) {
    return true;
  }

  const target = href.trim();
  return text === target || text === target.replace(/^https?:\/\//, "");
}
