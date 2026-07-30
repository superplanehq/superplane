import { buildSubtitle, buildExecutionSubtitle } from "../utils";

export const buildGitlabSubtitle = buildSubtitle;
export const buildGitlabExecutionSubtitle = buildExecutionSubtitle;

// Shorten a commit SHA for display, but leave expression placeholders (e.g.
// {{ event.data.checkout_sha }}) intact instead of slicing them mid-token.
export function shortSha(sha: string): string {
  return sha.includes("{{") ? sha : sha.slice(0, 8);
}
