import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

export interface AddIntakeTemplate {
  id: string;
  name: string;
  description: string;
  /** Optional integration icon. Letter glyph when omitted. */
  iconSrc?: string;
}

/**
 * Templates in the Add intake picker. Source-based intakes and a few
 * common improvement automations.
 */
export const ADD_INTAKE_TEMPLATES: AddIntakeTemplate[] = [
  {
    id: "github-issues",
    name: "GitHub issues",
    description: "Creates tasks from GitHub issues.",
    iconSrc: githubIcon,
  },
  {
    id: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Unresolved errors from production.",
    iconSrc: sentryIcon,
  },
  {
    id: "pagerduty-incidents",
    name: "PagerDuty incidents",
    description: "Firing incidents that need a work order.",
    iconSrc: pagerdutyIcon,
  },
  {
    id: "improve-ci-runtime",
    name: "Improve CI runtime",
    description: "Find slow jobs and cut pipeline wait time.",
  },
  {
    id: "improve-page-performance",
    name: "Improve page performance",
    description: "Track slow pages and open work to speed them up.",
  },
  {
    id: "flaky-tests",
    name: "Flaky tests",
    description: "Catch unstable tests and create fix work orders.",
  },
];

export function filterAddIntakeTemplates(query: string): AddIntakeTemplate[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return ADD_INTAKE_TEMPLATES;
  }
  return ADD_INTAKE_TEMPLATES.filter((template) => {
    const haystack = `${template.name} ${template.description}`.toLowerCase();
    return haystack.includes(needle);
  });
}
