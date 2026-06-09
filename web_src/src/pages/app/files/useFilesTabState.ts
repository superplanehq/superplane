import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { CANVAS_YAML_PATH } from "../lib/workflow-spec-paths";

const FILE_SEARCH_PARAM = "file";

function getDefaultInitialPath(allPaths: string[], generatedPaths: string[]): string | null {
  if (allPaths.includes(CANVAS_YAML_PATH)) return CANVAS_YAML_PATH;
  return generatedPaths[0] ?? allPaths[0] ?? null;
}

function resolveInitialPath(
  requestedPath: string | null,
  allPaths: string[],
  generatedPaths: string[],
  filesLoading: boolean,
): string | null {
  if (requestedPath && allPaths.includes(requestedPath)) return requestedPath;
  // Defer falling back to a default while files are still loading so that a
  // requested file (e.g. from the `file` URL param) that has not loaded yet
  // still wins once it arrives.
  if (requestedPath && filesLoading) return null;
  return getDefaultInitialPath(allPaths, generatedPaths);
}

export function useFilesTabState(allPaths: string[], generatedPaths: string[], filesLoading: boolean) {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedPath = searchParams.get(FILE_SEARCH_PARAM);
  const initialPath = resolveInitialPath(requestedPath, allPaths, generatedPaths, filesLoading);
  const hasAutoOpenedInitialFileRef = useRef(Boolean(initialPath));
  const [openTabs, setOpenTabs] = useState<string[]>(() => (initialPath ? [initialPath] : []));
  const [selectedPath, setSelectedPath] = useState<string | null>(() => initialPath);

  useEffect(() => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (selectedPath) {
          if (next.get(FILE_SEARCH_PARAM) === selectedPath) return current;
          next.set(FILE_SEARCH_PARAM, selectedPath);
        } else {
          // Keep the requested file in the URL while the file list is still
          // loading, otherwise a deep-linked file would be dropped before it
          // has a chance to be restored.
          if (filesLoading || !next.has(FILE_SEARCH_PARAM)) return current;
          next.delete(FILE_SEARCH_PARAM);
        }
        return next;
      },
      { replace: true },
    );
  }, [selectedPath, filesLoading, setSearchParams]);

  useEffect(() => {
    if (hasAutoOpenedInitialFileRef.current) return;

    const nextInitialPath = resolveInitialPath(requestedPath, allPaths, generatedPaths, filesLoading);
    if (!nextInitialPath) return;

    hasAutoOpenedInitialFileRef.current = true;
    setOpenTabs([nextInitialPath]);
    setSelectedPath(nextInitialPath);
  }, [allPaths, generatedPaths, requestedPath, filesLoading]);

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
