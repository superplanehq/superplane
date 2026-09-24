import { useRef } from "react";

import type { UseSpeechDictationResult } from "@/hooks/useSpeechDictation";
import { useSpokenPhraseDictation, type SpokenPhraseField } from "@/hooks/useSpokenPhraseDictation";

export type WorkOrderDictationField = "title" | "description";

export function useWorkOrderFieldDictation({
  title,
  description,
  maxTitleLength,
  maxDescriptionLength,
  onTitleChange,
  onDescriptionChange,
}: {
  title: string;
  description: string;
  maxTitleLength: number;
  maxDescriptionLength: number;
  onTitleChange: (next: string) => void;
  onDescriptionChange: (next: string) => void;
}): UseSpeechDictationResult & {
  rememberTitle: () => void;
  rememberDescription: () => void;
} {
  const lastFieldRef = useRef<WorkOrderDictationField>("description");
  const titleRef = useRef(title);
  const descriptionRef = useRef(description);
  titleRef.current = title;
  descriptionRef.current = description;

  const fieldRef = useRef<SpokenPhraseField>({
    key: lastFieldRef.current,
    getValue: () => descriptionRef.current,
    setValue: onDescriptionChange,
    get maxLength() {
      return maxDescriptionLength;
    },
  });
  fieldRef.current = {
    key: lastFieldRef.current,
    getValue: () => (lastFieldRef.current === "title" ? titleRef.current : descriptionRef.current),
    setValue: (next) => {
      if (lastFieldRef.current === "title") {
        titleRef.current = next;
        onTitleChange(next);
        return;
      }
      descriptionRef.current = next;
      onDescriptionChange(next);
    },
    get maxLength() {
      return lastFieldRef.current === "title" ? maxTitleLength : maxDescriptionLength;
    },
  };

  const { activateField, ...dictation } = useSpokenPhraseDictation(fieldRef);

  return {
    ...dictation,
    rememberTitle: () => {
      if (lastFieldRef.current === "title") {
        return;
      }
      lastFieldRef.current = "title";
      activateField("title");
    },
    rememberDescription: () => {
      if (lastFieldRef.current === "description") {
        return;
      }
      lastFieldRef.current = "description";
      activateField("description");
    },
  };
}
