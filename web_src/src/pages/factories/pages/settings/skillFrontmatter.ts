const NAME_MAX_LENGTH = 63;

export function sanitizeSkillCommandName(name: string): string {
  const stripped = name
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-+/, "")
    .replace(/-+$/, "");
  const startsWithLetter = stripped.replace(/^[^a-z]+/, "");
  return startsWithLetter.slice(0, NAME_MAX_LENGTH);
}

export function skillFrontmatterField(markdown: string, key: string): string | undefined {
  const match = markdown.match(/^---\s*\n([\s\S]*?)\n---/);
  if (!match) {
    return undefined;
  }
  const prefix = `${key}:`;
  const line = match[1]
    .split("\n")
    .map((entry) => entry.trim())
    .find((entry) => entry.startsWith(prefix));
  if (!line) {
    return undefined;
  }
  return unescapeYamlScalar(line.slice(prefix.length).trim());
}

export function setSkillFrontmatterFields(markdown: string, fields: Record<string, string>): string {
  const match = markdown.match(/^---\s*\n([\s\S]*?)\n---(\n[\s\S]*)?$/);
  if (!match) {
    const lines = Object.entries(fields).map(([key, value]) => `${key}: ${yamlScalar(value)}`);
    return `---\n${lines.join("\n")}\n---\n${markdown}`;
  }
  const body = match[2] ?? "\n";
  const lines = match[1].split("\n");
  for (const [key, value] of Object.entries(fields)) {
    const formatted = `${key}: ${yamlScalar(value)}`;
    const index = lines.findIndex((line) => line.trim().startsWith(`${key}:`));
    if (index >= 0) {
      lines[index] = formatted;
    } else {
      lines.push(formatted);
    }
  }
  return `---\n${lines.join("\n")}\n---${body.startsWith("\n") ? body : `\n${body}`}`;
}

export type SkillFieldsFromMarkdown = {
  title?: string;
  command?: string;
};

// Returns the Name field value (frontmatter title, else name) and the command
// (frontmatter name, else title, sanitized). A missing value is undefined.
export function skillFieldsFromMarkdown(markdown: string): SkillFieldsFromMarkdown {
  const name = skillFrontmatterField(markdown, "name")?.trim() ?? "";
  const title = skillFrontmatterField(markdown, "title")?.trim() ?? "";
  const displayTitle = title || name;
  const commandSource = name || title;
  return {
    title: displayTitle || undefined,
    command: commandSource ? sanitizeSkillCommandName(commandSource) : undefined,
  };
}

export function skillDisplayTitle(resource: { name?: string; markdown?: string }): string {
  const title = skillFrontmatterField(resource.markdown ?? "", "title")?.trim();
  if (title) {
    return title;
  }
  return resource.name?.trim() || "";
}

export function skillListDescription(resource: { markdown?: string; repository?: string; ref?: string }): string {
  const description = skillFrontmatterField(resource.markdown ?? "", "description")?.trim();
  if (description) {
    return description;
  }
  const repository = resource.repository?.trim() ?? "";
  if (!repository) {
    return "";
  }
  const ref = resource.ref?.trim();
  return ref ? `${repository}@${ref}` : repository;
}

function unescapeYamlScalar(value: string): string {
  const quote = value[0];
  if (quote === '"' || quote === "'") {
    const closing = findClosingQuote(value, quote);
    if (closing < 0) {
      return value;
    }
    const inner = value.slice(1, closing);
    return quote === "'" ? inner.replace(/''/g, "'") : inner;
  }
  // In YAML, " #" starts a comment in a plain scalar.
  const comment = value.search(/\s#/);
  return (comment >= 0 ? value.slice(0, comment) : value).trim();
}

function findClosingQuote(value: string, quote: string): number {
  for (let index = 1; index < value.length; index++) {
    if (quote === '"' && value[index] === "\\") {
      index++;
      continue;
    }
    if (value[index] !== quote) {
      continue;
    }
    if (quote === "'" && value[index + 1] === "'") {
      index++;
      continue;
    }
    return index;
  }
  return -1;
}

function yamlScalar(value: string): string {
  if (value === "") {
    return "";
  }
  if (/^[A-Za-z0-9][A-Za-z0-9 ._-]*$/.test(value)) {
    return value;
  }
  return JSON.stringify(value);
}
