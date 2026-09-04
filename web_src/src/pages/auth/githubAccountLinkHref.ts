export function getGitHubAccountLinkHref(): string {
  const params = new URLSearchParams({
    intent: "link",
    redirect: "/onboarding",
  });

  return `/auth/github?${params}`;
}
