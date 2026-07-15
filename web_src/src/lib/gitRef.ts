export type GitRefKind = "branch" | "tag" | "pull-request";

export function parseGitRef(ref?: string): { kind: GitRefKind; name: string } {
  const val = (ref || "").trim();
  if (val.startsWith("refs/heads/")) {
    return { kind: "branch", name: val.replace(/^refs\/heads\//, "") };
  }
  if (val.startsWith("ref/heads/")) {
    // Be tolerant of older placeholder without the trailing 's'
    return { kind: "branch", name: val.replace(/^ref\/heads\//, "") };
  }
  if (val.startsWith("refs/tags/")) {
    return { kind: "tag", name: val.replace(/^refs\/tags\//, "") };
  }
  if (val.startsWith("ref/tags/")) {
    return { kind: "tag", name: val.replace(/^ref\/tags\//, "") };
  }
  if (val.startsWith("refs/pull/")) {
    return { kind: "pull-request", name: normalizePullRequestName(val.replace(/^refs\/pull\//, "")) };
  }
  if (val.startsWith("ref/pull/")) {
    return { kind: "pull-request", name: normalizePullRequestName(val.replace(/^ref\/pull\//, "")) };
  }

  // Default to branch if unknown; keep whatever name is there
  return { kind: "branch", name: val };
}

export function buildGitRef(kind: GitRefKind, name: string): string {
  const sanitized = (name || "").trim();
  if (sanitized === "") return "";
  if (kind === "tag") return `refs/tags/${sanitized}`;
  if (kind === "pull-request") {
    const prNumber = normalizePullRequestName(sanitized);
    if (prNumber === "") return "";
    return `refs/pull/${prNumber}`;
  }
  return `refs/heads/${sanitized}`;
}

export function gitRefPlaceholder(kind: GitRefKind): string {
  if (kind === "tag") return "e.g. v1.0.0";
  if (kind === "pull-request") return "e.g. 123";
  return "e.g. main";
}

function normalizePullRequestName(name: string): string {
  return name.trim().replace(/\/(merge|head)$/i, "");
}
