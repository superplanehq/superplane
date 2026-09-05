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

export function hostedGitHubInstallRequested(metadata: unknown): boolean {
  if (!metadata || typeof metadata !== "object") {
    return false;
  }

  return (metadata as { installRequested?: unknown }).installRequested === true;
}

export function hostedGitHubInstallRequestedAccount(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const requested = (metadata as { installRequestedAccount?: unknown }).installRequestedAccount;
  if (typeof requested === "string" && requested !== "") {
    return requested;
  }

  const owner = (metadata as { owner?: unknown }).owner;
  return typeof owner === "string" ? owner : "";
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

/**
 * Binds a pending connection to an App installation without leaving the page.
 * The bind endpoint answers with a redirect on success, which fetch follows;
 * any error status means the bind did not happen.
 */
export async function bindHostedGitHubInstallation(state: string, installationId: string): Promise<void> {
  const response = await fetch(hostedGitHubBindPath(state, installationId), { credentials: "same-origin" });
  if (!response.ok) {
    throw new Error("Failed to connect the GitHub account");
  }
}

export function hostedGitHubInstallURL(slug: string, state: string): string {
  return `https://github.com/apps/${encodeURIComponent(slug)}/installations/new?state=${encodeURIComponent(state)}`;
}
