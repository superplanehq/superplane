/**
 * Known display names for integration types (by API name).
 * Used when the API label is missing so we still show "GitHub" not "Github".
 */
const INTEGRATION_TYPE_DISPLAY_NAMES: Record<string, string> = {
  github: "GitHub",
  gitlab: "GitLab",
  openai: "OpenAI",
  claude: "Claude",
  cursor: "Cursor",
  pagerduty: "PagerDuty",
  slack: "Slack",
  digitalocean: "DigitalOcean",
  discord: "Discord",
  datadog: "DataDog",
  cloudflare: "Cloudflare",
  semaphore: "Semaphore",
  rootly: "Rootly",
  statuspage: "Statuspage",
  daytona: "Daytona",
  dash0: "Dash0",
  aws: "AWS",
  smtp: "SMTP",
  sendgrid: "SendGrid",
  dockerhub: "DockerHub",
};

/**
 * Returns the display name for an integration type.
 * Always uses the known-names map when the (lowercase) name matches, so we never show "github" etc.
 * Otherwise uses the API label if it looks properly capitalized, or capitalizes the first letter.
 */
export function getIntegrationTypeDisplayName(label: string | undefined, name: string | undefined): string {
  // Use name for lookup; fall back to label so we still normalize when name is missing
  const key = (name ?? label)?.trim().toLowerCase();
  if (!key) return label?.trim() ?? "";

  // Always prefer known display name when we have one (e.g. "github" -> "GitHub")
  const known = INTEGRATION_TYPE_DISPLAY_NAMES[key];
  if (known) return known;

  let result: string;
  if (label?.trim()) {
    result = label.trim();
  } else {
    const fallback = name ?? label ?? "";
    result = fallback.charAt(0).toUpperCase() + fallback.slice(1);
  }
  // If result is still lowercase and we have a known name for it, use that (catches API returning "github" as label)
  const resultKey = result.toLowerCase();
  return INTEGRATION_TYPE_DISPLAY_NAMES[resultKey] ?? result;
}
