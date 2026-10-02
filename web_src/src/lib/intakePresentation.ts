import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import dependabotIcon from "@/assets/icons/integrations/dependabot.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import notionIcon from "@/assets/icons/integrations/notion.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

/** A screen that lists intakes or repository providers for a company. */
export type IntakeSurface = "addIntake" | "onboardingTickets" | "onboardingRepository";

export const INTAKE_SURFACE_ORDER: readonly IntakeSurface[] = [
  "addIntake",
  "onboardingTickets",
  "onboardingRepository",
];

export interface IntakePresentation {
  key: string;
  name: string;
  description: string;
  iconSrc?: string;
  /** Surfaces where the code supports this intake. */
  surfaces: readonly IntakeSurface[];
}

/**
 * Presentation for intakes that SuperPlane knows in code. The catalog decides
 * the state. An entry that an admin adds without code uses its category to
 * pick surfaces and shows as Coming soon.
 */
export const INTAKE_PRESENTATION: readonly IntakePresentation[] = [
  {
    key: "github-issues",
    name: "GitHub issues",
    description: "Creates tasks from GitHub issues.",
    iconSrc: githubIcon,
    surfaces: ["addIntake", "onboardingTickets"],
  },
  {
    key: "dependabot-alerts",
    name: "Dependabot alerts",
    description: "Creates tasks from Dependabot alerts.",
    iconSrc: dependabotIcon,
    surfaces: ["addIntake"],
  },
  {
    key: "jira-issues",
    name: "Jira issues",
    description: "Creates tasks from Jira issues.",
    iconSrc: jiraIcon,
    surfaces: ["addIntake", "onboardingTickets"],
  },
  {
    key: "linear-issues",
    name: "Linear issues",
    description: "Creates tasks from Linear issues.",
    iconSrc: linearIcon,
    surfaces: ["addIntake", "onboardingTickets"],
  },
  {
    key: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Creates tasks from Sentry exceptions.",
    iconSrc: sentryIcon,
    surfaces: ["addIntake"],
  },
  {
    key: "productive-tasks",
    name: "Productive tasks",
    description: "Creates tasks from Productive tasks.",
    iconSrc: productiveIcon,
    surfaces: ["addIntake"],
  },
  {
    key: "datadog",
    name: "Datadog errors",
    description: "Creates tasks from Datadog Error Tracking alerts.",
    iconSrc: datadogIcon,
    surfaces: ["addIntake"],
  },
  {
    key: "notion",
    name: "Notion",
    description: "Creates tasks from Notion pages.",
    iconSrc: notionIcon,
    surfaces: ["addIntake"],
  },
  {
    key: "pagerduty-incidents",
    name: "PagerDuty incidents",
    description: "Creates tasks from PagerDuty incidents.",
    iconSrc: pagerdutyIcon,
    surfaces: [],
  },
  {
    key: "github",
    name: "GitHub",
    description: "Connect GitHub to list repositories and open pull requests.",
    iconSrc: githubIcon,
    surfaces: ["onboardingRepository"],
  },
  {
    key: "gitlab",
    name: "GitLab",
    description: "Connect GitLab to list repositories and open merge requests.",
    iconSrc: gitlabIcon,
    surfaces: ["onboardingRepository"],
  },
  {
    key: "bitbucket",
    name: "Bitbucket",
    description: "Connect Bitbucket to list repositories and open pull requests.",
    iconSrc: bitbucketIcon,
    surfaces: ["onboardingRepository"],
  },
];

const CATEGORY_SURFACES: Record<string, readonly IntakeSurface[]> = {
  issue_tracking: ["addIntake", "onboardingTickets"],
  error_tracking: ["addIntake"],
  incident_management: ["addIntake"],
  security_alerts: ["addIntake"],
  repository_provider: ["onboardingRepository"],
};

export function findIntakePresentation(key: string): IntakePresentation | undefined {
  return INTAKE_PRESENTATION.find((presentation) => presentation.key === key);
}

/** Surfaces that list the entry. Code support wins over the category. */
export function intakeSurfaces(key: string, category: string): readonly IntakeSurface[] {
  const presentation = findIntakePresentation(key);
  if (presentation) {
    return presentation.surfaces;
  }
  return CATEGORY_SURFACES[category] ?? [];
}
