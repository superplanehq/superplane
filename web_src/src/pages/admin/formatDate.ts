export function formatDate(dateString?: string): string {
  if (!dateString) return "—";

  return new Date(dateString).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
