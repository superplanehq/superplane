import type { FactoriesFactoryAgentResource } from "@/api-client";

import { skillDisplayTitle, skillFrontmatterField } from "@/pages/factories/pages/settings/skillFrontmatter";

export interface SkillSlashCandidate {
  id: string;
  command: string;
  title: string;
  description: string;
}

export interface SkillSlashQuery {
  start: number;
  query: string;
}

const MAX_SKILL_SUGGESTIONS = 8;

export function skillSlashQueryAtCursor(value: string, cursor: number): SkillSlashQuery | null {
  const safeCursor = Math.max(0, Math.min(cursor, value.length));
  const before = value.slice(0, safeCursor);
  const slash = before.lastIndexOf("/");
  if (slash < 0 || !isSlashTrigger(value, slash)) {
    return null;
  }
  const query = before.slice(slash + 1);
  if (/\s/.test(query)) {
    return null;
  }
  return { start: slash, query };
}

export function filterSkillSlashCandidates(candidates: SkillSlashCandidate[], query: string): SkillSlashCandidate[] {
  const needle = query.trim().toLowerCase();
  const matches = needle ? candidates.filter((candidate) => skillSlashCandidateMatches(candidate, needle)) : candidates;
  return matches.slice(0, MAX_SKILL_SUGGESTIONS);
}

export function insertSkillSlashAtCursor(
  value: string,
  cursor: number,
  command: string,
): { value: string; cursor: number } {
  const inserted = `/${command} `;
  const query = skillSlashQueryAtCursor(value, cursor);
  if (!query) {
    return {
      value: value.slice(0, cursor) + inserted + value.slice(cursor),
      cursor: cursor + inserted.length,
    };
  }
  return {
    value: value.slice(0, query.start) + inserted + value.slice(cursor),
    cursor: query.start + inserted.length,
  };
}

export function skillSlashCandidatesFromResources(resources: FactoriesFactoryAgentResource[]): SkillSlashCandidate[] {
  return resources.flatMap((resource) => {
    if (resource.enabled === false) {
      return [];
    }
    const command = resource.name?.trim() ?? "";
    if (!command) {
      return [];
    }
    return [
      {
        id: resource.id || command,
        command,
        title: skillDisplayTitle(resource),
        description: skillFrontmatterField(resource.markdown ?? "", "description")?.trim() ?? "",
      },
    ];
  });
}

function isSlashTrigger(value: string, index: number): boolean {
  return index === 0 || /\s/.test(value[index - 1] ?? "");
}

function skillSlashCandidateMatches(candidate: SkillSlashCandidate, needle: string): boolean {
  return [candidate.command, candidate.title, candidate.description].some((part) =>
    part.toLowerCase().includes(needle),
  );
}
