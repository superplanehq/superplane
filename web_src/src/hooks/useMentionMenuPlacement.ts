import { computeMentionMenuPlacement, type MentionMenuPlacement } from "@/lib/aiBuilderMentionTypeahead";
import type { RefObject } from "react";
import { useCallback, useEffect, useLayoutEffect, useState } from "react";

export function useMentionMenuPlacement(
  mentionOpen: boolean,
  filteredCount: number,
  anchorRef: RefObject<HTMLDivElement | null>,
  aiInput: string,
): MentionMenuPlacement | null {
  const [mentionMenuPlacement, setMentionMenuPlacement] = useState<MentionMenuPlacement | null>(null);

  const updateMentionMenuPlacement = useCallback(() => {
    if (!mentionOpen || filteredCount === 0) {
      setMentionMenuPlacement(null);
      return;
    }
    const anchor = anchorRef.current;
    if (!anchor) {
      setMentionMenuPlacement(null);
      return;
    }
    setMentionMenuPlacement(computeMentionMenuPlacement(anchor.getBoundingClientRect()));
  }, [anchorRef, mentionOpen, filteredCount]);

  useLayoutEffect(() => {
    updateMentionMenuPlacement();
  }, [updateMentionMenuPlacement, aiInput, mentionOpen, filteredCount]);

  useEffect(() => {
    if (!mentionOpen || filteredCount === 0) {
      return;
    }
    const onReposition = () => updateMentionMenuPlacement();
    window.addEventListener("scroll", onReposition, true);
    window.addEventListener("resize", onReposition);
    return () => {
      window.removeEventListener("scroll", onReposition, true);
      window.removeEventListener("resize", onReposition);
    };
  }, [mentionOpen, filteredCount, updateMentionMenuPlacement]);

  return mentionMenuPlacement;
}
