/** Display label for the human user who created a service account (name only in the UI). */
export function formatServiceAccountCreatorLabel(serviceAccount: {
  createdByName?: string;
}): string | null {
  const name = serviceAccount.createdByName?.trim();
  if (!name) {
    return null;
  }
  return name;
}
