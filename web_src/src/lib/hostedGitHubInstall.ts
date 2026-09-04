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

export function hostedGitHubInstallURL(slug: string, state: string, targetId = ""): string {
  const path = `https://github.com/apps/${encodeURIComponent(slug)}/installations/new`;
  const params = new URLSearchParams();
  if (state) {
    params.set("state", state);
  }
  if (targetId) {
    params.set("target_id", targetId);
  }

  const query = params.toString();
  return query ? `${path}?${query}` : path;
}

export function isPersonalGitHubAccount(installation: PendingGitHubInstallation, personalLogin = ""): boolean {
  if (installation.accountType === "User") {
    return true;
  }

  return personalLogin !== "" && installation.accountLogin === personalLogin;
}

export function sortGitHubInstallations(
  installations: PendingGitHubInstallation[],
  personalLogin = "",
): PendingGitHubInstallation[] {
  return [...installations].sort((left, right) => {
    const leftPersonal = isPersonalGitHubAccount(left, personalLogin);
    const rightPersonal = isPersonalGitHubAccount(right, personalLogin);
    if (leftPersonal === rightPersonal) {
      return 0;
    }

    return leftPersonal ? -1 : 1;
  });
}

export function hostedGitHubUserId(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const id = (metadata as { githubUserId?: unknown }).githubUserId;
  return typeof id === "string" ? id : "";
}

export function hostedGitHubUserLogin(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const login = (metadata as { githubUserLogin?: unknown }).githubUserLogin;
  return typeof login === "string" ? login : "";
}
