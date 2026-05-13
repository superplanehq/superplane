import { useMemo } from "react";
import { FileText, FileCode, FileJson, FolderOpen } from "lucide-react";
import { cn } from "@/lib/utils";

interface FileInfo {
  path: string;
  size: number;
}

interface FileTreeProps {
  files: FileInfo[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
}

interface TreeNode {
  name: string;
  path: string;
  isDir: boolean;
  children: TreeNode[];
  size?: number;
}

function buildTree(files: FileInfo[]): TreeNode[] {
  const root: TreeNode[] = [];

  for (const file of files) {
    const parts = file.path.split("/");
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const name = parts[i];
      const isLast = i === parts.length - 1;
      const path = parts.slice(0, i + 1).join("/");

      let existing = current.find((n) => n.name === name);
      if (!existing) {
        existing = {
          name,
          path,
          isDir: !isLast,
          children: [],
          size: isLast ? file.size : undefined,
        };
        current.push(existing);
      }
      current = existing.children;
    }
  }

  // Sort: dirs first, then alphabetical
  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
    for (const node of nodes) {
      sortNodes(node.children);
    }
  };
  sortNodes(root);
  return root;
}

function getIcon(name: string, isDir: boolean) {
  if (isDir) return <FolderOpen className="h-3.5 w-3.5 text-sky-500" />;
  if (name.endsWith(".yaml") || name.endsWith(".yml")) return <FileCode className="h-3.5 w-3.5 text-orange-500" />;
  if (name.endsWith(".json")) return <FileJson className="h-3.5 w-3.5 text-yellow-600" />;
  if (name.endsWith(".md")) return <FileText className="h-3.5 w-3.5 text-blue-500" />;
  return <FileText className="h-3.5 w-3.5 text-slate-400" />;
}

export function FileTree({ files, selectedPath, onSelect }: FileTreeProps) {
  const tree = useMemo(() => buildTree(files), [files]);

  return (
    <div className="py-1">
      {tree.map((node) => (
        <TreeNodeView
          key={node.path}
          node={node}
          depth={0}
          selectedPath={selectedPath}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

function TreeNodeView({
  node,
  depth,
  selectedPath,
  onSelect,
}: {
  node: TreeNode;
  depth: number;
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  const isSelected = node.path === selectedPath;

  return (
    <>
      <button
        type="button"
        className={cn(
          "flex w-full items-center gap-1.5 px-3 py-1 text-left text-xs hover:bg-slate-100",
          isSelected && "bg-slate-200 font-medium",
          node.isDir && "text-slate-600",
          !node.isDir && "text-slate-700",
        )}
        style={{ paddingLeft: `${12 + depth * 12}px` }}
        onClick={() => {
          if (!node.isDir) {
            onSelect(node.path);
          }
        }}
      >
        {getIcon(node.name, node.isDir)}
        <span className="truncate">{node.name}</span>
      </button>
      {node.isDir &&
        node.children.map((child) => (
          <TreeNodeView
            key={child.path}
            node={child}
            depth={depth + 1}
            selectedPath={selectedPath}
            onSelect={onSelect}
          />
        ))}
    </>
  );
}
