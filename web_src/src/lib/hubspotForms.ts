import type { SignupWaitlistConfig } from "@/lib/signupWaitlistConfig";

type HubSpotSubmissionContext = {
  hutk?: string;
  pageName?: string;
  pageUri?: string;
};

type HubSpotSubmissionPayload = {
  fields: Array<{
    name: string;
    value: string;
  }>;
  context?: HubSpotSubmissionContext;
};

const defaultHubSpotFormsHost = "api.hsforms.com";

const getHubSpotFormsHost = (region?: string) => {
  if (!region || region === "na1") {
    return defaultHubSpotFormsHost;
  }

  return `api-${region}.hsforms.com`;
};

const hubSpotSubmissionURL = ({ portalID, formID, region }: SignupWaitlistConfig) =>
  `https://${getHubSpotFormsHost(region)}/submissions/v3/integration/submit/${encodeURIComponent(portalID)}/${encodeURIComponent(formID)}`;

const getCookieValue = (name: string) => {
  const prefix = `${name}=`;
  const cookie = document.cookie.split("; ").find((part) => part.startsWith(prefix));
  return cookie ? decodeURIComponent(cookie.slice(prefix.length)) : undefined;
};

const getSubmissionContext = (): HubSpotSubmissionContext => {
  const hutk = getCookieValue("hubspotutk");

  return {
    ...(hutk ? { hutk } : {}),
    pageName: document.title || undefined,
    pageUri: window.location.href,
  };
};

export const submitSignupWaitlistEmail = async (config: SignupWaitlistConfig, email: string) => {
  const payload: HubSpotSubmissionPayload = {
    fields: [{ name: "email", value: email }],
    context: getSubmissionContext(),
  };

  const response = await fetch(hubSpotSubmissionURL(config), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(`HubSpot form submission failed with status ${response.status}`);
  }
};
