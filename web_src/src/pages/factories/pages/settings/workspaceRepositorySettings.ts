import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export type WorkspaceRepositoryAccess = "write" | "read";

export type WorkspaceRepositorySettings = {
  ref: string;
  access: WorkspaceRepositoryAccess;
};

export function isWorkspaceMainRepository(name: string, appRepository: string | undefined): boolean {
  const main = appRepository?.trim().toLowerCase() ?? "";
  if (!main) return false;
  return name.trim().toLowerCase() === main;
}

export function workspaceRepositoryAccess(access: string | null | undefined): WorkspaceRepositoryAccess {
  return access === "write" ? "write" : "read";
}

export function workspaceRepositorySummary(settings: WorkspaceRepositorySettings, defaultBranch = ""): string {
  const branch = settings.ref || defaultBranch || AGENT_RESOURCES_COPY.defaultBranch;
  const access = settings.access === "write" ? AGENT_RESOURCES_COPY.accessWrite : AGENT_RESOURCES_COPY.accessRead;
  return `${branch} · ${access}`;
}
