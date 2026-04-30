/** Human-readable label for the user who created a service account (API: createdBy + createdByUser). */
export function serviceAccountCreatorLabel(serviceAccount: {
  createdBy?: string;
  createdByUser?: { id?: string; name?: string };
}): string {
  const trimmedName = serviceAccount.createdByUser?.name?.trim();
  if (trimmedName) {
    return trimmedName;
  }
  if (serviceAccount.createdByUser?.id || serviceAccount.createdBy) {
    return "Unknown";
  }
  return "—";
}
