import { resolveIcon } from "@/lib/utils";
import React from "react";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import awsLambdaIcon from "@/assets/icons/integrations/aws.lambda.svg";
import awsEcsIcon from "@/assets/icons/integrations/aws.ecs.svg";
import circleciIcon from "@/assets/icons/integrations/circleci.svg";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import awsRoute53Icon from "@/assets/icons/integrations/aws.route53.svg";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import digitaloceanIcon from "@/assets/icons/integrations/digitalocean.svg";
import discordIcon from "@/assets/icons/integrations/discord.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import smtpIcon from "@/assets/icons/integrations/smtp.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import renderIcon from "@/assets/icons/integrations/render.svg";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import hetznerIcon from "@/assets/icons/integrations/hetzner.svg";
import statuspageIcon from "@/assets/icons/integrations/statuspage.svg";

/** Integration type name (e.g. "github") → logo src. Used for Settings tab and header. */
export const INTEGRATION_APP_LOGO_MAP: Record<string, string> = {
  aws: awsIcon,
  bitbucket: bitbucketIcon,
  circleci: circleciIcon,
  cloudflare: cloudflareIcon,
  dash0: dash0Icon,
  datadog: datadogIcon,
  daytona: daytonaIcon,
  digitalocean: digitaloceanIcon,
  discord: discordIcon,
  github: githubIcon,
  gitlab: gitlabIcon,
  hetzner: hetznerIcon,
  grafana: grafanaIcon,
  jira: jiraIcon,
  openai: openAiIcon,
  "open-ai": openAiIcon,
  claude: claudeIcon,
  cursor: cursorIcon,
  pagerduty: pagerDutyIcon,
  rootly: rootlyIcon,
  semaphore: SemaphoreLogo,
  slack: slackIcon,
  smtp: smtpIcon,
  sendgrid: sendgridIcon,
  prometheus: prometheusIcon,
  render: renderIcon,
  dockerhub: dockerIcon,
  statuspage: statuspageIcon,
};

/** Block name first part (e.g. "github") or compound (e.g. aws.lambda) → logo src for header. */
export const APP_LOGO_MAP: Record<string, string | Record<string, string>> = {
  bitbucket: bitbucketIcon,
  circleci: circleciIcon,
  cloudflare: cloudflareIcon,
  dash0: dash0Icon,
  datadog: datadogIcon,
  daytona: daytonaIcon,
  digitalocean: digitaloceanIcon,
  discord: discordIcon,
  github: githubIcon,
  gitlab: gitlabIcon,
  hetzner: hetznerIcon,
  grafana: grafanaIcon,
  jira: jiraIcon,
  openai: openAiIcon,
  "open-ai": openAiIcon,
  claude: claudeIcon,
  cursor: cursorIcon,
  pagerduty: pagerDutyIcon,
  rootly: rootlyIcon,
  semaphore: SemaphoreLogo,
  slack: slackIcon,
  smtp: smtpIcon,
  sendgrid: sendgridIcon,
  prometheus: prometheusIcon,
  render: renderIcon,
  dockerhub: dockerIcon,
  statuspage: statuspageIcon,
  aws: {
    cloudwatch: awsCloudwatchIcon,
    codeArtifact: awsCodeArtifactIcon,
    ecr: awsEcrIcon,
    lambda: awsLambdaIcon,
    ec2: awsEc2Icon,
    route53: awsRoute53Icon,
    ecs: awsEcsIcon,
    sns: awsSnsIcon,
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
  // Use main AWS icon for aws.* components without a dedicated sub-icon mapping.
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
