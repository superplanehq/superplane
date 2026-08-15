import type { CanvasesReplayInput, CanvasesResolvedReplayInputStatus } from "@/api-client";
import { validateReplayPayload } from "./validateReplayPayload";

export type ReplayInputDraft = {
  sourceNodeId?: string;
  label: string;
  editedText: string;
  status?: CanvasesResolvedReplayInputStatus;
};

export type ReplayInputFieldResult =
  | { valid: true; payload: Record<string, unknown> }
  | { valid: false; error: string; kind: "invalid-json" | "missing" };

export type ReplayLaunchRequest = {
  inputs: CanvasesReplayInput[];
};

export type ReplayLaunchValidation = {
  valid: boolean;
  fields: ReplayInputFieldResult[];
  request: ReplayLaunchRequest | null;
  launchError?: string;
};

const NOTHING_REPLAYABLE = "Every input's source no longer feeds this node, so there is nothing to replay";

function isEmptyPayload(payload: Record<string, unknown>): boolean {
  return Object.keys(payload).length === 0;
}

function validateField(draft: ReplayInputDraft): ReplayInputFieldResult {
  // Detached inputs are never sent, so their content can never block the launch.
  if (draft.status === "STATUS_DETACHED") {
    return { valid: true, payload: {} };
  }

  const parsed = validateReplayPayload(draft.editedText);
  if (!parsed.valid) {
    return { valid: false, error: `Invalid JSON for ${draft.label}: ${parsed.error}`, kind: "invalid-json" };
  }

  // A missing or expired source starts out empty; launching with it still empty would
  // silently replay with nothing, so refuse client-side rather than sending a blank payload.
  if ((draft.status === "STATUS_MISSING" || draft.status === "STATUS_EXPIRED") && isEmptyPayload(parsed.value)) {
    return {
      valid: false,
      error: `Missing payload for ${draft.label} — fill it in before launching the replay`,
      kind: "missing",
    };
  }

  return { valid: true, payload: parsed.value };
}

function buildRequest(
  drafts: ReplayInputDraft[],
  fields: Array<{ valid: true; payload: Record<string, unknown> }>,
): ReplayLaunchRequest | null {
  // Filter before reading a field's payload, so a detached input's placeholder payload
  // is never pulled into `live` in the first place — exclusion doesn't hinge on a later step.
  const live = drafts
    .map((draft, index) => ({ draft, index }))
    .filter(({ draft }) => draft.status !== "STATUS_DETACHED")
    .map(({ draft, index }) => ({ draft, payload: fields[index].payload }));

  if (live.length === 0) {
    return null;
  }

  // Always the attributed `inputs`, even for a single input: the server resolves
  // from history whenever inputs is empty and a source execution is given, which
  // would replace the payload the user just edited with the stored one.
  return { inputs: live.map(({ draft, payload }) => ({ payload, sourceNodeId: draft.sourceNodeId })) };
}

export function validateReplayLaunch(drafts: ReplayInputDraft[]): ReplayLaunchValidation {
  const fields = drafts.map(validateField);

  if (fields.some((field) => !field.valid)) {
    return { valid: false, fields, request: null };
  }

  const validFields = fields as Array<{ valid: true; payload: Record<string, unknown> }>;
  const request = buildRequest(drafts, validFields);
  if (!request) {
    return { valid: false, fields, request: null, launchError: NOTHING_REPLAYABLE };
  }

  return { valid: true, fields, request };
}
