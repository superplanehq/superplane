export function splitSentryIssueTitle(title?: string): { title?: string; prefix?: string } {
  if (!title) {
    return {};
  }

  const trimmedTitle = title.trim();
  if (!trimmedTitle) {
    return {};
  }

  const separatorIndex = trimmedTitle.indexOf(":");
  if (separatorIndex <= 0 || separatorIndex >= trimmedTitle.length - 1) {
    return { title: trimmedTitle };
  }

  const prefix = trimmedTitle.slice(0, separatorIndex).trim();
  const suffix = trimmedTitle.slice(separatorIndex + 1).trim();

  if (!prefix || !suffix) {
    return { title: trimmedTitle };
  }

  return {
    title: suffix,
    prefix,
  };
}

export function addDetail(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = value;
}

export function addFormattedTimestamp(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = new Date(value).toLocaleString();
}

export function getProjectLabel(issue?: { project?: { name?: string; slug?: string } }) {
  return issue?.project?.name || issue?.project?.slug;
}
