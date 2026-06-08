import { useCallback, useEffect, useRef, useState } from "react";

export function useFilesTabState(initialPath: string | null, allPaths: string[], generatedPaths: string[]) {
  const hasAutoOpenedInitialFileRef = useRef(Boolean(initialPath));
  const [openTabs, setOpenTabs] = useState<string[]>(() => (initialPath ? [initialPath] : []));
  const [selectedPath, setSelectedPath] = useState<string | null>(() => initialPath);

  useEffect(() => {
    if (hasAutoOpenedInitialFileRef.current) return;

    const nextInitialPath = generatedPaths[0] ?? allPaths[0];
    if (!nextInitialPath) return;

    hasAutoOpenedInitialFileRef.current = true;
    setOpenTabs([nextInitialPath]);
    setSelectedPath(nextInitialPath);
  }, [allPaths, generatedPaths]);

  useEffect(() => {
    const allPathSet = new Set(allPaths);
    setOpenTabs((current) => current.filter((path) => allPathSet.has(path)));
    setSelectedPath((current) => (current && allPathSet.has(current) ? current : null));
  }, [allPaths]);

  const openFile = useCallback((path: string) => {
    setOpenTabs((current) => (current.includes(path) ? current : [...current, path]));
    setSelectedPath(path);
  }, []);

  const closeTab = useCallback((path: string) => {
    setOpenTabs((current) => {
      const nextTabs = current.filter((tabPath) => tabPath !== path);
      setSelectedPath((selected) => {
        if (selected !== path) return selected;
        const closedIndex = current.indexOf(path);
        return nextTabs[Math.min(closedIndex, nextTabs.length - 1)] ?? null;
      });
      return nextTabs;
    });
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.key.toLowerCase() !== "w") return;
      if (!selectedPath) return;

      event.preventDefault();
      closeTab(selectedPath);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [closeTab, selectedPath]);

  return { openTabs, selectedPath, openFile, closeTab };
}
