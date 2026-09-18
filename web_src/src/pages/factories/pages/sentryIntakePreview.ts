import { SENTRY_INTAKE_SEED_SIZE, SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

export type SentryIntakePreviewRow = {
  id: string;
  title: string;
};

const SENTRY_SHORT_ID_SEPARATOR = " · ";

export function sentryIntakePreviewTitle(name: string): string {
  const trimmed = name.trim();
  const index = trimmed.indexOf(SENTRY_SHORT_ID_SEPARATOR);
  if (index <= 0) {
    return trimmed;
  }
  const title = trimmed.slice(index + SENTRY_SHORT_ID_SEPARATOR.length).trim();
  return title || trimmed;
}

export function sentryIntakePreviewRows(issueNames: string[]): SentryIntakePreviewRow[] {
  const titles = issueNames.map((title) => sentryIntakePreviewTitle(title)).filter(Boolean);
  const source = titles.length > 0 ? titles : [...SENTRY_INTAKE_SETUP_COPY.exampleIssues];
  return source.slice(0, SENTRY_INTAKE_SEED_SIZE).map((title, index) => ({
    id: `issue-${index + 1}`,
    title,
  }));
}

export function sentryIntakePreviewCaption(hasProject: boolean): string {
  if (!hasProject) {
    return SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaptionNoProject;
  }
  return SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaption;
}
