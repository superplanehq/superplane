export type PendingGitHubInstallation = {
  id: string;
  accountLogin: string;
  accountType?: string;
  repositories: PendingGitHubRepository[];
};

export type PendingGitHubRepository = {
  id: string;
  name: string;
  url?: string;
};

export type PendingGitHubInstallRequest = {
  id?: string;
  accountLogin: string;
  requesterLogin?: string;
  createdAt?: string;
};

export function pendingGitHubInstallRequests(metadata: unknown): PendingGitHubInstallRequest[] {
  if (!metadata || typeof metadata !== "object") return [];

  const raw = (metadata as { installRequests?: unknown }).installRequests;
  if (Array.isArray(raw)) {
    return raw.flatMap((item) => {
      if (!item || typeof item !== "object") return [];
      const row = item as {
        id?: unknown;
        accountLogin?: unknown;
        requesterLogin?: unknown;
        createdAt?: unknown;
      };
      const id = typeof row.id === "number" ? String(row.id) : row.id;
      return [
        {
          ...(typeof id === "string" && id !== "" ? { id } : {}),
          accountLogin: typeof row.accountLogin === "string" ? row.accountLogin.trim() : "",
          ...(typeof row.requesterLogin === "string" && row.requesterLogin !== ""
            ? { requesterLogin: row.requesterLogin }
            : {}),
          ...(typeof row.createdAt === "string" && row.createdAt !== "" ? { createdAt: row.createdAt } : {}),
        },
      ];
    });
  }

  if ((metadata as { installRequested?: unknown }).installRequested !== true) return [];
  return [{ accountLogin: hostedGitHubInstallRequestedAccount(metadata) }];
}

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

    const row = item as { id?: unknown; accountLogin?: unknown; accountType?: unknown; repositories?: unknown };
    const id = typeof row.id === "number" ? String(row.id) : row.id;
    if (typeof id !== "string" || id === "" || typeof row.accountLogin !== "string" || row.accountLogin === "") {
      return [];
    }

    return [
      {
        id,
        accountLogin: row.accountLogin,
        accountType: typeof row.accountType === "string" ? row.accountType : undefined,
        repositories: pendingGitHubRepositories(row.repositories),
      },
    ];
  });
}

function pendingGitHubRepositories(value: unknown): PendingGitHubRepository[] {
  if (!Array.isArray(value)) return [];

  return value.flatMap((item) => {
    if (!item || typeof item !== "object") return [];
    const row = item as { id?: unknown; name?: unknown; url?: unknown };
    const id = typeof row.id === "number" ? String(row.id) : row.id;
    if (typeof id !== "string" || id === "" || typeof row.name !== "string" || row.name === "") return [];
    return [{ id, name: row.name, ...(typeof row.url === "string" && row.url !== "" ? { url: row.url } : {}) }];
  });
}

export function hostedGitHubInstallRequested(metadata: unknown): boolean {
  if (!metadata || typeof metadata !== "object") {
    return false;
  }

  return (
    (metadata as { installRequested?: unknown }).installRequested === true ||
    pendingGitHubInstallRequests(metadata).length > 0
  );
}

export function hostedGitHubInstallRequestedAccount(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const requested = (metadata as { installRequestedAccount?: unknown }).installRequestedAccount;
  if (typeof requested === "string" && requested !== "") {
    return requested;
  }

  return "";
}

export function hostedGitHubState(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const state = (metadata as { state?: unknown }).state;
  return typeof state === "string" ? state : "";
}

/**
 * GitHub login of the member who authorized this connect. The account
 * picker uses it so the user can see which GitHub session the listed
 * installations belong to.
 */
export function hostedGitHubStartedByLogin(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const login = (metadata as { startedByGitHubLogin?: unknown }).startedByGitHubLogin;
  return typeof login === "string" ? login : "";
}

export function hostedGitHubAppSlug(metadata: unknown): string {
  if (!metadata || typeof metadata !== "object") {
    return "";
  }

  const app = (metadata as { githubApp?: { slug?: unknown } }).githubApp;
  return typeof app?.slug === "string" ? app.slug : "";
}

export function hostedGitHubBindPath(): string {
  return "/api/v1/github/app/bind";
}

/**
 * Binds a pending connection to an App installation without leaving the page.
 * The bind endpoint returns no content on success and an error status when
 * the bind did not happen.
 */
export async function bindHostedGitHubInstallation(
  state: string,
  installationId: string,
  repositoryId: string,
): Promise<void> {
  const body = new URLSearchParams({ state, installation_id: installationId, repository_id: repositoryId });
  const response = await fetch(hostedGitHubBindPath(), {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });
  if (!response.ok) {
    throw new Error("Failed to connect the GitHub account");
  }
}

export function hostedGitHubInstallURL(slug: string, state: string): string {
  return `https://github.com/apps/${encodeURIComponent(slug)}/installations/new?state=${encodeURIComponent(state)}`;
}
