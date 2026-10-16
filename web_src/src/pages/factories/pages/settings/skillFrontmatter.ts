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
  const trimmed = value.trim();
  const quoted = quotedYamlScalar(trimmed);
  if (quoted !== undefined) {
    return quoted;
  }
  return plainYamlScalar(trimmed);
}

function quotedYamlScalar(value: string): string | undefined {
  const quote = value[0];
  if (quote === "'") {
    return singleQuotedYamlScalar(value);
  }
  if (quote === '"') {
    return doubleQuotedYamlScalar(value);
  }
  return undefined;
}

function singleQuotedYamlScalar(value: string): string | undefined {
  let decoded = "";
  for (let index = 1; index < value.length; index++) {
    const char = value[index];
    if (char !== "'") {
      decoded += char;
      continue;
    }
    if (value[index + 1] === "'") {
      decoded += "'";
      index += 1;
      continue;
    }
    return decoded;
  }
  return undefined;
}

function doubleQuotedYamlScalar(value: string): string | undefined {
  const end = closingDoubleQuote(value);
  if (end < 0) {
    return undefined;
  }
  try {
    const parsed: unknown = JSON.parse(value.slice(0, end + 1));
    return typeof parsed === "string" ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function closingDoubleQuote(value: string): number {
  let escaped = false;
  for (let index = 1; index < value.length; index++) {
    const char = value[index];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (char === "\\") {
      escaped = true;
      continue;
    }
    if (char === '"') {
      return index;
    }
  }
  return -1;
}

function plainYamlScalar(value: string): string {
  const commentStart = value.search(/(?:^|\s)#/);
  if (commentStart < 0) {
    return value;
  }
  return value.slice(0, commentStart).trim();
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
