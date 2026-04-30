/** Display label for the human user who created a service account (API fields). */
export function formatServiceAccountCreatorLabel(serviceAccount: {
  createdByName?: string;
  createdByEmail?: string;
}): string | null {
  const name = serviceAccount.createdByName?.trim();
  const email = serviceAccount.createdByEmail?.trim();
  if (name && email) {
    return `${name} (${email})`;
  }
  if (name) {
    return name;
  }
  if (email) {
    return email;
  }
  return null;
}
