import { useCallback, useEffect, useRef, useState } from "react";
import { FileTree } from "./FileTree";
import { FileViewer } from "./FileViewer";
import { CloneDropdown } from "./CloneDropdown";
import { Loader2, GitBranch } from "lucide-react";
import { cn } from "@/lib/utils";
import { useHeaderActionSlotSetter } from "@/ui/CanvasPage/HeaderActionSlotContext";

import type { NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";

interface RepoTabProps {
  canvasId: string;
  canvasName: string;
  organizationId: string;
  nodeRefs?: NodeChipContext;
}

interface FileInfo {
  path: string;
  size: number;
}

interface CommitInfo {
  sha: string;
  message: string;
  date: string;
  author: string;
}


const REPO_SIDEBAR_MIN_WIDTH = 200;
const REPO_SIDEBAR_MAX_WIDTH = 480;
const REPO_SIDEBAR_DEFAULT_WIDTH = 260;
const REPO_SIDEBAR_WIDTH_KEY = "repo-sidebar-width";

export function RepoTab({ canvasId, canvasName, organizationId, nodeRefs }: RepoTabProps) {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [commit, setCommit] = useState<CommitInfo | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Resizable sidebar
  const sidebarRef = useRef<HTMLDivElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const stored = localStorage.getItem(REPO_SIDEBAR_WIDTH_KEY);
    return stored ? Number(stored) : REPO_SIDEBAR_DEFAULT_WIDTH;
  });
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;
      const left = sidebarRef.current?.getBoundingClientRect()?.left ?? 0;
      const w = Math.max(REPO_SIDEBAR_MIN_WIDTH, Math.min(REPO_SIDEBAR_MAX_WIDTH, e.clientX - left));
      setSidebarWidth(w);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => setIsResizing(false), []);

  useEffect(() => { localStorage.setItem(REPO_SIDEBAR_WIDTH_KEY, String(sidebarWidth)); }, [sidebarWidth]);

  useEffect(() => {
    if (!isResizing) return;
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "ew-resize";
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  // Register Clone button in the header action slot
  const setHeaderActionNode = useHeaderActionSlotSetter();
  const slug = files.length > 0 ? (commit as CommitInfo | null) : null; // just need to trigger re-render
  useEffect(() => {
    if (!setHeaderActionNode) return;
    setHeaderActionNode(
      <CloneDropdown
        repoUrl={`${window.location.origin}/git/${canvasName.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "")}`}
        organizationId={organizationId}
        canvasId={canvasId}
      />
    );
    return () => setHeaderActionNode(null);
  }, [setHeaderActionNode, canvasName]);

  const fetchFiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/repo/${canvasId}/files`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Failed to load files: ${res.status}`);
      const data = await res.json();
      setFiles(data.files || []);
      setCommit(data.commit || null);

      // Auto-select docs/README.md or first .md file
      const fileList: FileInfo[] = data.files || [];
      const readme = fileList.find((f: FileInfo) => f.path === "docs/README.md");
      const rootReadme = fileList.find((f: FileInfo) => f.path === "README.md");
      const firstMd = fileList.find((f: FileInfo) => f.path.endsWith(".md"));
      setSelectedPath(readme?.path || rootReadme?.path || firstMd?.path || fileList[0]?.path || null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [canvasId]);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-slate-500">{error}</p>
          <p className="mt-1 text-xs text-slate-400">Make sure the canvas has a git repository.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0">
      {/* Left sidebar: file tree */}
      <div
        ref={sidebarRef}
        className="relative flex shrink-0 flex-col border-r border-slate-200 bg-white"
        style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      >
        <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
          <div className="flex items-center gap-1.5 text-sm font-medium text-gray-700">
            <GitBranch className="h-3.5 w-3.5" />
            main
          </div>
        </div>
        <div className="flex-1 overflow-auto">
          <FileTree
            files={files}
            selectedPath={selectedPath}
            onSelect={setSelectedPath}
          />
        </div>
        {commit ? (
          <div className="flex shrink-0 items-center gap-2 border-t border-slate-200 bg-slate-50 px-3 py-1.5">
            <div className="min-w-0 flex-1">
              <p className="truncate text-[11px] text-gray-500" title={`${commit.sha.slice(0, 7)} ${commit.message}`}>
                <code className="font-mono text-[10px] text-gray-400">{commit.sha.slice(0, 7)}</code>
                {" · "}{commit.message}
              </p>
              <p className="text-[10px] text-gray-400">
                {commit.author} · {formatRelativeDate(commit.date)}
              </p>
            </div>
          </div>
        ) : null}

        {/* Resize handle */}
        <div
          onMouseDown={handleMouseDown}
          className={cn(
            "absolute right-0 top-0 bottom-0 z-30 flex w-4 cursor-ew-resize items-center justify-center transition-colors hover:bg-gray-100 group",
            isResizing && "bg-blue-50",
          )}
          style={{ marginRight: "-8px" }}
          aria-label="Resize sidebar"
          role="separator"
        >
          <div
            className={cn(
              "h-14 w-1 rounded-full bg-gray-300 transition-colors group-hover:bg-gray-800",
              isResizing && "bg-blue-500",
            )}
          />
        </div>
      </div>

      {/* Right panel: file viewer */}
      <div className="flex-1 min-w-0 overflow-auto">
        {selectedPath ? (
          <FileViewer
            canvasId={canvasId}
            path={selectedPath}
            nodeRefs={nodeRefs}
            onSaved={fetchFiles}
          />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-slate-400">
            Select a file to view
          </div>
        )}
      </div>
    </div>
  );
}

function formatRelativeDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}
