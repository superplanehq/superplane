export function parseGitHubRepoParam(raw: string | null | undefined): { owner: string; repo: string } | null {
  let trimmed = raw?.trim();
  if (!trimmed) {
    return null;
  }

  trimmed = trimmed.replace(/^https?:\/\//i, "");
  trimmed = trimmed.replace(/\.git$/i, "");
  trimmed = trimmed.replace(/^\/+|\/+$/g, "");

  if (/^github\.com\//i.test(trimmed)) {
    trimmed = trimmed.slice("github.com/".length);
  } else {
    try {
      const url = new URL(`https://${trimmed}`);
      if (url.hostname.toLowerCase() === "github.com") {
        trimmed = url.pathname.replace(/^\/+|\/+$/g, "");
      }
    } catch {
      // Keep trimmed as-is for bare owner/repo paths.
    }
  }

  const parts = trimmed.split("/").filter(Boolean);
  if (parts.length !== 2) {
    return null;
  }

  return {
    owner: parts[0],
    repo: parts[1],
  };
}

export function formatGitHubRepoParam(owner: string, repo: string): string {
  return `github.com/${owner}/${repo}`;
}
