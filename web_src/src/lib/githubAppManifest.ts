import { followBrowserAction } from "@/lib/browserAction";
import { getResponseErrorMessage } from "@/lib/errors";

type GitHubAppManifestResponse = {
  url?: string;
  method?: string;
  form?: Record<string, string>;
};

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
