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
