export type PendingGitHubInstallation = {
  id: string;
  accountLogin: string;
  accountType?: string;
};

export function pendingGitHubInstallations(metadata: unknown): PendingGitHubInstallation[] {
  if (!metadata || typeof metadata !== "object") {
    return [];
  }

  const raw = (metadata as { pendingInstallations?: unknown }).pendingInstallations;
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw.flatMap((item) => {
    if (!item || typeof item !== "object") {
      return [];
    }

    const row = item as { id?: unknown; accountLogin?: unknown; accountType?: unknown };
    const id = typeof row.id === "number" ? String(row.id) : row.id;
    if (typeof id !== "string" || id === "" || typeof row.accountLogin !== "string" || row.accountLogin === "") {
      return [];
    }

    return [
      {
        id,
        accountLogin: row.accountLogin,
        accountType: typeof row.accountType === "string" ? row.accountType : undefined,
      },
    ];
  });
}

export function hostedGitHubState(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const state = (metadata as { state?: unknown }).state;
  return typeof state === "string" ? state : "";
}

export function hostedGitHubAppSlug(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const app = (metadata as { githubApp?: { slug?: unknown } }).githubApp;
  return typeof app?.slug === "string" ? app.slug : "";
}

export function hostedGitHubBindPath(state: string, installationId: string): string {
  const params = new URLSearchParams({
    state,
    installation_id: installationId,
  });
  return `/api/v1/github/app/bind?${params.toString()}`;
}

export function hostedGitHubInstallURL(slug: string, state: string): string {
  return `https://github.com/apps/${encodeURIComponent(slug)}/installations/new?state=${encodeURIComponent(state)}`;
}
