import type { Meta, StoryObj } from "@storybook/react-vite";

import { AppPage } from "./index";
import { AppPageHarness } from "./__fixtures__/AppPageHarness";
import { consoleFixtures } from "./__fixtures__/consoleFixtures";

/**
 * Full-`AppPage` stories that render the **console tab** of four production
 * apps against captured fixture data. Each fixture is a sanitized snapshot of
 * a real app on app.superplane.com (canvas describe, runs, memory, versions,
 * and the `console.yaml` that drives the dashboard) so we can iterate on the
 * console UI against realistic layouts without needing network access or a
 * running backend.
 *
 * PII was scrubbed from the captures: every email is remapped to a
 * deterministic `user-<n>@example.com`, every UUID (org, canvas, runs,
 * events, users) is remapped to a deterministic fake derived from its hash
 * (preserving referential integrity between fields), and GitHub avatar URLs
 * point at the public octocat. Contributor display names are kept because
 * the dashboards render them and they are public on this repository.
 *
 * The harness itself (fetch override, memory router, React Query wiring) is
 * shared with the graph-view stories in `AppPage.stories.tsx` — see
 * `__fixtures__/AppPageHarness.tsx` for the rationale.
 */
const meta = {
  title: "Pages/AppPage/Console",
  component: AppPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof AppPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const consoleQuery = "view=console";

const { superplaneSaas, prRiskReview, docsReviewer, superplaneRelease } = consoleFixtures;

/**
 * SuperPlane SaaS — production deployment pipeline console.
 *
 * Shows the "Currently in Production" markdown card, a "Deployments" runs
 * table, a "Deployment duration" chart, and two KPI cards (average duration
 * and count over the last 7 days).
 */
export const SuperPlaneSaas: Story = {
  name: "SuperPlane SaaS",
  render: () => <AppPageHarness query={consoleQuery} fixture={superplaneSaas} />,
};

/**
 * PR Risk Review — pull request risk assessment console.
 *
 * Shows the "How review works" README, a manual "Check PR" trigger card, a
 * "Recent checks" table with per-PR risk scores, and quick-answer explainer
 * cards.
 */
export const PrRiskReview: Story = {
  name: "PR Risk Review",
  render: () => <AppPageHarness query={consoleQuery} fixture={prRiskReview} />,
};

/**
 * Docs Reviewer — documentation change review console.
 *
 * Shows the review workflow README, a manual review trigger, and the recent
 * docs review history.
 */
export const DocsReviewer: Story = {
  name: "Docs Reviewer",
  render: () => <AppPageHarness query={consoleQuery} fixture={docsReviewer} />,
};

/**
 * SuperPlane Release — release management console.
 *
 * Shows the release status, in-flight releases, and release history for the
 * SuperPlane open-source distribution.
 */
export const SuperPlaneRelease: Story = {
  name: "SuperPlane Release",
  render: () => <AppPageHarness query={consoleQuery} fixture={superplaneRelease} />,
};
