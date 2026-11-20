export const REVIEW_FINDINGS_BODY_NODE_ID = "findings-body";
export const REVIEW_FINDINGS_COMMENT_NODE_ID = "findings-comment";

export interface ReviewFindingsSettings {
  publish: boolean;
  body: boolean;
  comment: boolean;
}

type Configuration = Record<string, unknown> | undefined;

const destinations = [
  {
    nodeId: REVIEW_FINDINGS_BODY_NODE_ID,
    setting: "body",
    generationKey: "reviewFindingsBodyActivationGeneration",
  },
  {
    nodeId: REVIEW_FINDINGS_COMMENT_NODE_ID,
    setting: "comment",
    generationKey: "reviewFindingsCommentActivationGeneration",
  },
] as const;

// Absent values use the template defaults: off, with PR body selected.
export function readReviewFindings(configuration: Configuration): ReviewFindingsSettings {
  return {
    publish: configuration?.publishReviewFindings === true,
    body: configuration?.reviewFindingsBody !== false,
    comment: configuration?.reviewFindingsComment === true,
  };
}

export function writeReviewFindings(configuration: Record<string, unknown>, settings: ReviewFindingsSettings) {
  return {
    ...configuration,
    publishReviewFindings: settings.publish,
    reviewFindingsBody: settings.body,
    reviewFindingsComment: settings.comment,
  };
}

// Publication needs a destination. When publication is off, the saved
// destinations stay unchanged so the user can turn it on again.
export function reviewFindingsMissingDestination(settings: ReviewFindingsSettings): boolean {
  return settings.publish && !settings.body && !settings.comment;
}

export interface ReviewFindingsDestination {
  nodeId: string;
  enabled: boolean;
  activationGeneration: number;
}

// Each saved off-to-on change of a destination starts a new activation. Attempts
// from an earlier activation cannot write after the destination is enabled again.
export function reviewFindingsActivation(saved: Configuration, draft: Configuration) {
  const before = readReviewFindings(saved);
  const after = readReviewFindings(draft);
  const generations: Record<string, number> = {};
  const nodes: ReviewFindingsDestination[] = destinations.map((destination) => {
    const wasEnabled = before.publish && before[destination.setting];
    const enabled = after.publish && after[destination.setting];
    const savedGeneration = Number(saved?.[destination.generationKey]);
    const current = Number.isInteger(savedGeneration) && savedGeneration > 0 ? savedGeneration : 1;
    const activationGeneration = current + (enabled && !wasEnabled ? 1 : 0);
    generations[destination.generationKey] = activationGeneration;
    return { nodeId: destination.nodeId, enabled, activationGeneration };
  });
  return { generations, nodes };
}
