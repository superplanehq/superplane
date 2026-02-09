import { resolveIcon } from "@/lib/utils";
import React from "react";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import awsLambdaIcon from "@/assets/icons/integrations/aws.lambda.svg";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import discordIcon from "@/assets/icons/integrations/discord.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import smtpIcon from "@/assets/icons/integrations/smtp.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";

/** Integration type name (e.g. "github") → logo src. Used for Settings tab and header. */
export const INTEGRATION_APP_LOGO_MAP: Record<string, string> = {
  aws: awsIcon,
  cloudflare: cloudflareIcon,
  dash0: dash0Icon,
  datadog: datadogIcon,
  daytona: daytonaIcon,
  discord: discordIcon,
  github: githubIcon,
  jira: jiraIcon,
  openai: openAiIcon,
  "open-ai": openAiIcon,
  claude: claudeIcon,
  pagerduty: pagerDutyIcon,
  rootly: rootlyIcon,
  semaphore: SemaphoreLogo,
  slack: slackIcon,
  smtp: smtpIcon,
  sendgrid: sendgridIcon,
};

/** Block name first part (e.g. "github") or compound (e.g. aws.lambda) → logo src for header. */
export const APP_LOGO_MAP: Record<string, string | Record<string, string>> = {
  cloudflare: cloudflareIcon,
  dash0: dash0Icon,
  datadog: datadogIcon,
  daytona: daytonaIcon,
  discord: discordIcon,
  github: githubIcon,
  jira: jiraIcon,
  openai: openAiIcon,
  "open-ai": openAiIcon,
  claude: claudeIcon,
  pagerduty: pagerDutyIcon,
  rootly: rootlyIcon,
  semaphore: SemaphoreLogo,
  slack: slackIcon,
  sendgrid: sendgridIcon,
  aws: {
    lambda: awsLambdaIcon,
    sns: awsIcon,
  },
};

/**
 * Returns logo src for an integration type (e.g. "github" → github icon).
 * Use this for consistent integration icons in Settings tab and header.
 */
export function getIntegrationIconSrc(integrationName: string | undefined): string | undefined {
  if (!integrationName) return undefined;
  const key = integrationName.toLowerCase();
  return INTEGRATION_APP_LOGO_MAP[key];
}

/**
 * Returns logo src for the component header from block name (e.g. "github.runWorkflow" or "aws.lambda").
 * For AWS, uses the main AWS icon when no nested icon exists (e.g. aws.runFunction) instead of Lucide fallback.
 */
export function getHeaderIconSrc(blockName: string | undefined): string | undefined {
  if (!blockName) return undefined;
  const nameParts = blockName.split(".");
  const first = nameParts[0];
  if (!first) return undefined;
  const appLogo = APP_LOGO_MAP[first];
  if (typeof appLogo === "string") return appLogo;
  if (nameParts[1] && appLogo) {
    const nested = appLogo[nameParts[1]];
    if (nested) return nested;
  }
  // AWS has a nested map (lambda only); use main AWS icon for other aws.* components
  if (first === "aws") return getIntegrationIconSrc("aws");
  return undefined;
}

const DEFAULT_ICON_SIZE = 16;

interface IntegrationIconProps {
  integrationName: string | undefined;
  /** Fallback Lucide icon slug when no custom logo exists */
  iconSlug?: string;
  className?: string;
  size?: number;
}

/**
 * Renders the integration's custom logo when available, otherwise a Lucide icon.
 * Use next to integration names (Settings tab) and in the component header for consistency.
 */
export function IntegrationIcon({
  integrationName,
  iconSlug,
  className = "h-4 w-4",
  size = DEFAULT_ICON_SIZE,
}: IntegrationIconProps): React.ReactElement {
  const logoSrc = getIntegrationIconSrc(integrationName);
  if (logoSrc) {
    return (
      <span className={`inline-block flex-shrink-0 ${className}`}>
        <img src={logoSrc} alt="" className="h-full w-full object-contain" />
      </span>
    );
  }
  const IconComponent = resolveIcon(iconSlug);
  return <IconComponent className={className} size={size} />;
}
