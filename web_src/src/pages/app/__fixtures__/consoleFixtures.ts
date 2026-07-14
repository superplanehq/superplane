import docsReviewerFixture from "./console/docsReviewer.json";
import prRiskReviewFixture from "./console/prRiskReview.json";
import superplaneReleaseFixture from "./console/superplaneRelease.json";
import superplaneSaasFixture from "./console/superplaneSaas.json";
import superplaneSaasReadme from "./repository/superplaneSaas.README.md?raw";
import type { CanvasAppFixture } from "./handlers";

// Storybook's static indexer cannot parse `.json` imports from `.stories.tsx`
// files (it runs acorn over direct story imports). Keep JSON imports here so
// the console page stories only reference this module.
export const consoleFixtures = {
  superplaneSaas: {
    ...(superplaneSaasFixture as CanvasAppFixture),
    repositoryFileContents: {
      "README.md": superplaneSaasReadme,
    },
  } satisfies CanvasAppFixture,
  prRiskReview: prRiskReviewFixture as CanvasAppFixture,
  docsReviewer: docsReviewerFixture as CanvasAppFixture,
  superplaneRelease: superplaneReleaseFixture as CanvasAppFixture,
} as const;
