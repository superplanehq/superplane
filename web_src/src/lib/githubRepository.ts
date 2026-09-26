/** GitHub "Code security and analysis" settings for a `owner/repo` backlog repository. */
export function githubRepositorySecurityAnalysisUrl(repository: string): string | undefined {
  const trimmed = repository.trim();
  const match = /^([a-zA-Z0-9_.-]+)\/([a-zA-Z0-9_.-]+)$/.exec(trimmed);
  if (!match) {
    return undefined;
  }

  return `https://github.com/${match[1]}/${match[2]}/settings/security_analysis`;
}
