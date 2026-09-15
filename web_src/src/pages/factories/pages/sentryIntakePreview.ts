import { SENTRY_INTAKE_SEED_SIZE, SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

export type SentryIntakePreviewRow = {
  id: string;
  title: string;
};

export function sentryIntakePreviewRows(issueNames: string[]): SentryIntakePreviewRow[] {
  const titles = issueNames.map((title) => title.trim()).filter(Boolean);
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
