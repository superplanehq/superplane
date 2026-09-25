import { useMemo } from "react";

import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { filterSkillSlashCandidates, skillSlashCandidatesFromResources } from "@/lib/skillSlash";

export function useSkillSlashCandidates(
  organizationId: string | undefined,
  factoryId: string | undefined,
  filter: string,
  enabled: boolean,
) {
  const skills = useFactoryAgentResources(organizationId ?? "", factoryId ?? "", "KIND_SKILL", enabled);
  return useMemo(
    () => filterSkillSlashCandidates(skillSlashCandidatesFromResources(skills.data ?? []), filter),
    [filter, skills.data],
  );
}
