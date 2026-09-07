import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_EMAIL,
  STORYBOOK_ME_USER_NAME,
} from "@/pages/factories/__fixtures__/factoryPageResponses";

import type { FixtureResult } from "./handlers";

const STORYBOOK_GITHUB_SIGN_IN_EMAIL = "john.doe@users.noreply.github.com";

export function storybookAccountProviders(email: string) {
  return [
    { provider: "github", username: "ada", email: STORYBOOK_GITHUB_SIGN_IN_EMAIL },
    { provider: "google", username: email, email },
  ];
}

export function createStorybookAccountState(orgId: string) {
  let email = STORYBOOK_ME_USER_EMAIL;
  let name = STORYBOOK_ME_USER_NAME;
  const providers = storybookAccountProviders(STORYBOOK_ME_USER_EMAIL);

  const json = () => ({
    id: "storybook-user",
    email,
    name,
    organization_id: orgId,
    avatar_url: STORYBOOK_ME_USER_AVATAR_URL,
    has_password: true,
    providers: providers.map((provider) => ({ ...provider })),
  });

  const applyDisconnect = (providerName: string): FixtureResult => {
    const index = providers.findIndex((provider) => provider.provider === providerName);
    if (index === -1) {
      return { json: {} };
    }
    const [removed] = providers.splice(index, 1);
    const current = email.trim().toLowerCase();
    const next = providers.find((provider) => {
      const nextEmail = (provider.email ?? "").trim().toLowerCase();
      return nextEmail !== "" && nextEmail !== current;
    });
    if ((removed.email ?? "").trim().toLowerCase() === current && next?.email) {
      email = next.email;
    }
    return { json: json() };
  };

  const applyPatch = (body: unknown) => {
    if (!body || typeof body !== "object") {
      return;
    }
    const patch = body as { name?: unknown; email?: unknown };
    if (typeof patch.name === "string" && patch.name.trim()) {
      name = patch.name.trim();
    }
    if (typeof patch.email === "string" && patch.email.trim()) {
      email = patch.email.trim();
    }
  };

  return {
    match(url: URL, method: string, body: unknown): FixtureResult {
      const providerMatch = /^\/account\/providers\/([^/]+)$/.exec(url.pathname);
      if (providerMatch) {
        return method === "DELETE" ? applyDisconnect(providerMatch[1]) : null;
      }
      if (url.pathname !== "/account" || (method !== "GET" && method !== "PATCH")) {
        return null;
      }
      if (method === "PATCH") {
        applyPatch(body);
      }
      return { json: json() };
    },
  };
}
