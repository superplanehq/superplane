const RUN_TITLE_FIELD_NAME = "customName";

export const RUN_TITLE_EXCLUDED_SUGGESTIONS = ["$", "previous"];

export function getRunTitlePresentation(fieldName: string | undefined, isEnabled: boolean) {
  if (fieldName !== RUN_TITLE_FIELD_NAME) {
    return null;
  }

  return {
    label: "Customize run title",
    previewLabel: "Preview title",
    description: isEnabled
      ? "Set the title for runs started by this trigger. Use root().data to reference fields from the trigger event."
      : "This trigger starts a run when an event arrives. By default, SuperPlane names the run from the event payload.",
  };
}
