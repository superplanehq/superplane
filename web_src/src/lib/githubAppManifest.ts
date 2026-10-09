import { followBrowserAction } from "@/lib/browserAction";
import { getResponseErrorMessage } from "@/lib/errors";

type GitHubAppManifestResponse = {
  url?: string;
  method?: string;
  form?: Record<string, string>;
};

export type GitHubLoginClientState = "missing" | "needs_client" | "ready";

export async function fetchGitHubLoginClient(): Promise<GitHubLoginClientState> {
  const response = await fetch("/api/v1/github/app/login", { credentials: "include" });
  if (!response.ok) {
    throw new Error(await getResponseErrorMessage(response, "SuperPlane could not read the GitHub App"));
  }
  const body = (await response.json()) as { state?: string };
  if (body.state === "needs_client" || body.state === "ready") return body.state;
  return "missing";
}

export async function saveGitHubLoginClient(clientId: string, clientSecret: string): Promise<void> {
  const response = await fetch("/api/v1/github/app/login", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ clientId, clientSecret }),
  });
  if (!response.ok) {
    throw new Error(await getResponseErrorMessage(response, "SuperPlane could not save the GitHub login"));
  }
}

export async function startPublicGitHubAppCreate(returnPath: string): Promise<boolean> {
  const response = await fetch(`/api/v1/github/app/manifest?return_to=${encodeURIComponent(returnPath)}`, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(await getResponseErrorMessage(response, "SuperPlane could not start GitHub App setup"));
  }
  const body = (await response.json()) as GitHubAppManifestResponse;
  return followBrowserAction({
    method: body.method,
    url: body.url,
    formFields: body.form,
  });
}
