import { useEffect, useState, type KeyboardEvent, type MutableRefObject } from "react";

import { useSkillSlashCandidates } from "@/hooks/useSkillSlashCandidates";
import { insertSkillSlashAtCursor, skillSlashQueryAtCursor } from "@/lib/skillSlash";

import { SkillSlashMenu } from "./SkillSlashMenu";

export function SkillSlashFieldOverlay({
  organizationId,
  factoryId,
  value,
  cursor,
  onInsert,
  keyboardRef,
}: {
  organizationId?: string;
  factoryId: string;
  value: string;
  cursor: number;
  onInsert: (next: { value: string; cursor: number }) => void;
  keyboardRef?: MutableRefObject<((event: KeyboardEvent) => boolean) | null>;
}) {
  const [dismissed, setDismissed] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(0);
  const query = skillSlashQueryAtCursor(value, cursor);
  const candidates = useSkillSlashCandidates(
    organizationId,
    factoryId,
    query?.query ?? "",
    Boolean(query) && !dismissed,
  );

  useEffect(() => {
    setDismissed(false);
  }, [query?.start]);

  useEffect(() => {
    setHighlightIndex(0);
  }, [query?.query, query?.start]);

  useEffect(() => {
    if (!keyboardRef) {
      return;
    }
    if (!query || dismissed || candidates.length === 0) {
      keyboardRef.current = null;
      return;
    }
    const handleKeyboard = (event: KeyboardEvent): boolean => {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setHighlightIndex((index) => (index < candidates.length - 1 ? index + 1 : 0));
        return true;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setHighlightIndex((index) => (index > 0 ? index - 1 : candidates.length - 1));
        return true;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        const selected = candidates[highlightIndex] ?? candidates[0];
        if (selected) {
          onInsert(insertSkillSlashAtCursor(value, cursor, selected.command));
        }
        return true;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setDismissed(true);
        return true;
      }
      return false;
    };
    keyboardRef.current = handleKeyboard;
    return () => {
      if (keyboardRef.current === handleKeyboard) {
        keyboardRef.current = null;
      }
    };
  }, [query, dismissed, candidates, highlightIndex, onInsert, keyboardRef, value, cursor]);

  return (
    <SkillSlashMenu
      candidates={query && !dismissed ? candidates : []}
      highlightIndex={highlightIndex}
      onHighlight={setHighlightIndex}
      onSelect={(candidate) => onInsert(insertSkillSlashAtCursor(value, cursor, candidate.command))}
    />
  );
}
