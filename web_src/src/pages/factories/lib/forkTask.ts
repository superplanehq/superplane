import { SPEC_ARTIFACT_NAME } from "./intentDocument";
import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "./workOrderArtifact";

export const FORK_TASK_COPY = {
  menu: "Fork task",
  title: "Fork task",
  helper: "The source task stays as it is.",
  intake: "Copy request",
  intakeHelper: "Creates a new draft. Planning starts if it is on.",
  plan: "Copy plan",
  planHelper: "Creates a draft with the plan. Start it when ready.",
  noPlan: "This task has no plan.",
  permission: "You do not have permission to fork this task.",
  error: "SuperPlane could not fork this task.",
  submit: "Fork task",
  cancel: "Cancel",
} as const;

export function taskHasSpec(artifacts?: Array<{ data?: unknown }>): boolean {
  return (artifacts ?? []).some((artifact) => {
    const data = toArtifactDataRecord(artifact.data);
    const name = extractArtifactName(data) ?? extractArtifactTitle(data) ?? "";
    const body = extractArtifactMarkdownBody(data)?.trim() ?? "";
    return name === SPEC_ARTIFACT_NAME && body.length > 0;
  });
}
