import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import notionIcon from "@/assets/icons/integrations/notion.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

export interface AddIntakeTemplate {
  id: string;
  name: string;
  description: string;
  /** Optional integration icon. Letter glyph when omitted. */
  iconSrc?: string;
  /** True when SuperPlane does not create this intake yet. */
  soon?: boolean;
}

export const ADD_INTAKE_COPY = {
  pickerTitle: "Add intake",
  pickerDescription: "Choose a source for new backlog tasks.",
  sourceTaken: "Already set up.",
  comingSoon: "Coming soon.",
} as const;

/**
 * Sources in the Add intake picker. Live sources create an intake.
 * Coming-soon sources stay visible and disabled.
 */
export const ADD_INTAKE_TEMPLATES: AddIntakeTemplate[] = [
  {
    id: "github-issues",
    name: "GitHub issues",
    description: "Creates tasks from GitHub issues.",
    iconSrc: githubIcon,
  },
  {
    id: "jira-issues",
    name: "Jira issues",
    description: "Creates tasks from Jira issues.",
    iconSrc: jiraIcon,
  },
  {
    id: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Adds the 10 newest unresolved issues. New issues become tasks.",
    iconSrc: sentryIcon,
  },
  {
    id: "datadog",
    name: "DataDog",
    description: "Creates tasks from DataDog monitors.",
    iconSrc: datadogIcon,
    soon: true,
  },
  {
    id: "notion",
    name: "Notion",
    description: "Creates tasks from Notion pages.",
    iconSrc: notionIcon,
    soon: true,
  },
];
