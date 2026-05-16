export function oauthDraftFromAllowedProviders(providers: string[] | undefined) {
  const list = providers ?? [];
  if (list.length === 0) {
    return { restrict: false, github: true, google: true };
  }
  return { restrict: true, github: list.includes("github"), google: list.includes("google") };
}

export function oauthProvidersListEqual(a: string[], b: string[]) {
  if (a.length !== b.length) {
    return false;
  }
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((v, i) => v === sortedB[i]);
}

export function oauthSavedPolicySummary(providers: string[] | undefined) {
  const saved = providers ?? [];
  if (saved.length === 0) {
    return "Saved: any OAuth provider can complete pending email invitations.";
  }
  const labels = saved.map((p) => (p === "github" ? "GitHub" : p === "google" ? "Google" : p));
  const joined = labels.length === 2 ? `${labels[0]} and ${labels[1]}` : labels.join(", ");
  return `Saved: only ${joined} can complete pending email invitations.`;
}

export function oauthProvidersToSave(restrict: boolean, github: boolean, google: boolean): string[] {
  if (!restrict) {
    return [];
  }
  const out: string[] = [];
  if (github) {
    out.push("github");
  }
  if (google) {
    out.push("google");
  }
  return out;
}
