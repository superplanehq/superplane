export type JiraCompletionColumnValue = {
  jiraMoveOnComplete: boolean;
  jiraCompletionColumn: string;
};

export function preferredJiraCompletionColumn(names: string[], current: string): string {
  const trimmed = current.trim();
  if (trimmed && names.some((name) => name === trimmed)) {
    return trimmed;
  }
  const done = names.find((name) => name.toLowerCase() === "done");
  return done ?? names[0] ?? "";
}
