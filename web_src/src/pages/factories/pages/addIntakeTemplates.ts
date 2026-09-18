import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import notionIcon from "@/assets/icons/integrations/notion.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { FEATURE_FACTORY_JIRA_INTAKE, FEATURE_FACTORY_SENTRY_INTAKE } from "@/lib/experimentalFeatures";

export interface AddIntakeTemplate {
  id: string;
  name: string;
  description: string;
  /** Optional integration icon. Letter glyph when omitted. */
  iconSrc?: string;
  /** True when SuperPlane does not create this intake yet. */
  soon?: boolean;
  /** Organization experimental feature that must be on for this source to be live. */
  featureId?: string;
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
    description: "Adds the 10 newest unresolved issues. New issues become tasks.",
    iconSrc: jiraIcon,
    featureId: FEATURE_FACTORY_JIRA_INTAKE,
  },
  {
    id: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Adds the 10 newest unresolved issues. New issues become tasks.",
    iconSrc: sentryIcon,
    featureId: FEATURE_FACTORY_SENTRY_INTAKE,
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

export function isAddIntakeSoon(template: AddIntakeTemplate, hasFeature: (featureId: string) => boolean): boolean {
  if (template.soon) {
    return true;
  }
  if (!template.featureId) {
    return false;
  }
  return !hasFeature(template.featureId);
}

/** Catalog with Coming soon applied when the organization feature is off. */
export function addIntakeTemplatesForOrg(hasFeature: (featureId: string) => boolean): AddIntakeTemplate[] {
  return ADD_INTAKE_TEMPLATES.map((template) =>
    isAddIntakeSoon(template, hasFeature) ? { ...template, soon: true } : template,
  );
}
