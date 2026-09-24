import { useLayoutEffect, useRef } from "react";

const streamMemory = new Map<string, string>();

/** Test helper. Strict Mode and popup remounts share this memory. */
export function resetStreamMemoryForTests(): void {
  streamMemory.clear();
}

/**
 * Play after this pane has already seen a value. The previous value is stored
 * in an effect so React Strict Mode cannot flip play back to false.
 */
export function useStreamOnUpdate(value: string, memoryKey?: string, ready = true): boolean {
  const seen = useRef<string | undefined>(undefined);
  if (ready && memoryKey && seen.current === undefined && streamMemory.has(memoryKey)) {
    seen.current = streamMemory.get(memoryKey);
  }
  const play = ready && seen.current !== undefined && seen.current !== value;
  useLayoutEffect(() => {
    if (!ready) {
      return;
    }
    seen.current = value;
    if (memoryKey) {
      streamMemory.set(memoryKey, value);
    }
  }, [memoryKey, ready, value]);
  return play;
}
