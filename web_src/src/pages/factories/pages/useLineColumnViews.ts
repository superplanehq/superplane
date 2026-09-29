import { useEffect, useRef, useState } from "react";

import { defaultLineColumnView, type LineColumnViewChoice } from "../lib/lineColumnView";

export function useLineColumnViews(lineId: string | undefined) {
  const [scopeId, setScopeId] = useState(lineId);
  const [choices, setChoices] = useState<Record<string, LineColumnViewChoice>>({});

  if (lineId !== scopeId) {
    setScopeId(lineId);
    setChoices({});
  }

  const choiceFor = (key: string): LineColumnViewChoice => choices[key] ?? defaultLineColumnView();

  const setChoice = (key: string, choice: LineColumnViewChoice) => {
    setChoices((current) => ({ ...current, [key]: choice }));
  };

  return { choiceFor, setChoice };
}

export function useExhaustColumnPages(
  page: { hasMore: boolean; isLoading: boolean; isError: boolean; onLoadMore: () => void },
  enabled: boolean,
) {
  const onLoadMore = useRef(page.onLoadMore);
  onLoadMore.current = page.onLoadMore;

  useEffect(() => {
    if (!enabled || !page.hasMore || page.isLoading || page.isError) {
      return;
    }
    onLoadMore.current();
  }, [enabled, page.hasMore, page.isLoading, page.isError]);
}
