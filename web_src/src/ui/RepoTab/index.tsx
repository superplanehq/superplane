import { useCallback, useEffect, useState } from "react";
import { FileTree } from "./FileTree";
import { FileViewer } from "./FileViewer";
import { Loader2, GitBranch } from "lucide-react";

import type { NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";

interface RepoTabProps {
  canvasId: string;
  canvasName: string;
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


export function RepoTab({ canvasId, canvasName, nodeRefs }: RepoTabProps) {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [commit, setCommit] = useState<CommitInfo | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
      <div className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-slate-50">
        <div className="border-b border-slate-200 px-3 py-2">
          <div className="flex items-center gap-1.5 text-xs font-medium text-slate-600">
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
          <div className="border-t border-slate-200 px-3 py-2">
            <p className="truncate text-[10px] text-slate-400" title={`${commit.sha.slice(0, 7)} ${commit.message}`}>
              {commit.sha.slice(0, 7)} · {commit.message}
            </p>
            <p className="text-[10px] text-slate-400">
              {commit.author} · {formatRelativeDate(commit.date)}
            </p>
          </div>
        ) : null}
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
