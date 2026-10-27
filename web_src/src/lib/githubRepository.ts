import { githubRepoWebUrl } from "./githubHost";

/** GitHub "Code security and analysis" settings for a `owner/repo` backlog repository. */
export function githubRepositorySecurityAnalysisUrl(repository: string, host?: string | null): string | undefined {
  const repoUrl = githubRepoWebUrl(repository, host);
  if (!repoUrl) {
    return undefined;
  }
  return `${repoUrl}/settings/security_analysis`;
}
