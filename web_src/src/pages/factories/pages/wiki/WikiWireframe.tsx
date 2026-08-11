import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ChevronRight, FileText, Folder } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { factoryPageSubtitleClassName, factoryPageTitleClassName } from "../factoryPageLayoutStyles";
import {
  buildWikiTree,
  wikiFolderPaths,
  WIKI_DOCUMENTS_DEFAULT,
  WIKI_DOCUMENTS_REFRESHED,
  type WikiDocument,
  type WikiTreeNode,
} from "./wikiMocks";

export type WikiWireframeProps = {
  initialDocuments?: WikiDocument[];
  /** Story helper — start in edit mode when a doc is selected. */
  startEditing?: boolean;
};

function TreeNode({
  node,
  depth,
  path,
  selectedId,
  expanded,
  onToggle,
  onSelect,
}: {
  node: WikiTreeNode;
  depth: number;
  path: string;
  selectedId: string | null;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  onSelect: (id: string) => void;
}) {
  if (node.kind === "file") {
    const active = selectedId === node.document.id;
    return (
      <li>
        <button
          type="button"
          title={node.document.path}
          onClick={() => onSelect(node.document.id)}
          className={cn(
            "flex w-full items-center gap-1.5 rounded-md py-1 pr-2 text-left text-[13px]",
            active ? "bg-sidebar-accent text-foreground" : "text-foreground/80 hover:bg-sidebar-accent/70",
          )}
          style={{ paddingLeft: 8 + depth * 12 }}
        >
          <span className="size-3.5 shrink-0" aria-hidden />
          <FileText className="size-3.5 shrink-0 opacity-70" strokeWidth={1.75} />
          <span className="truncate">{node.name}</span>
        </button>
      </li>
    );
  }

  const open = expanded.has(path);
  return (
    <li>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => onToggle(path)}
        className="flex w-full items-center gap-1.5 rounded-md py-1 pr-2 text-left text-[13px] text-foreground/80 hover:bg-sidebar-accent/70"
        style={{ paddingLeft: 8 + depth * 12 }}
      >
        <ChevronRight
          className={cn("size-3.5 shrink-0 opacity-70 transition-transform", open && "rotate-90")}
          strokeWidth={1.75}
        />
        <Folder className="size-3.5 shrink-0 opacity-70" strokeWidth={1.75} />
        <span className="truncate">{node.name}</span>
      </button>
      {open ? (
        <ul>
          {node.children.map((child) => {
            const childPath = child.kind === "folder" ? `${path}/${child.name}` : path;
            return (
              <TreeNode
                key={child.kind === "folder" ? `folder:${childPath}` : child.document.id}
                node={child}
                depth={depth + 1}
                path={child.kind === "folder" ? childPath : path}
                selectedId={selectedId}
                expanded={expanded}
                onToggle={onToggle}
                onSelect={onSelect}
              />
            );
          })}
        </ul>
      ) : null}
    </li>
  );
}

/**
 * Storybook-only wiki wireframe (v3 parity). Not mounted on the app `/wiki` route.
 */
export function WikiWireframe({ initialDocuments = WIKI_DOCUMENTS_DEFAULT, startEditing = false }: WikiWireframeProps) {
  const [documents, setDocuments] = useState<WikiDocument[]>(initialDocuments);
  const [selectedId, setSelectedId] = useState<string | null>(initialDocuments[0]?.id ?? null);
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const [treeReady, setTreeReady] = useState(false);
  const [editing, setEditing] = useState(startEditing);
  const [draftTitle, setDraftTitle] = useState("");
  const [draftContent, setDraftContent] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const tree = useMemo(() => buildWikiTree(documents), [documents]);
  const selected = documents.find((doc) => doc.id === selectedId) ?? null;

  useEffect(() => {
    if (treeReady || tree.length === 0) return;
    setExpanded(new Set(wikiFolderPaths(tree)));
    setTreeReady(true);
  }, [tree, treeReady]);

  useEffect(() => {
    if (!selected) {
      setDraftTitle("");
      setDraftContent("");
      return;
    }
    setDraftTitle(selected.title);
    setDraftContent(selected.content);
    if (!startEditing) setEditing(false);
  }, [selected, startEditing]);

  function toggleFolder(path: string) {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }

  function saveEdit() {
    if (!selected) return;
    setDocuments((current) =>
      current.map((doc) =>
        doc.id === selected.id ? { ...doc, title: draftTitle.trim() || doc.title, content: draftContent } : doc,
      ),
    );
    setEditing(false);
  }

  function refreshKnowledge() {
    if (refreshing) return;
    setRefreshing(true);
    window.setTimeout(() => {
      setDocuments(WIKI_DOCUMENTS_REFRESHED);
      setSelectedId(WIKI_DOCUMENTS_REFRESHED[0]?.id ?? null);
      setExpanded(new Set());
      setTreeReady(false);
      setEditing(false);
      setRefreshing(false);
    }, 800);
  }

  return (
    <div className="flex h-screen min-h-0 flex-col" data-testid="wiki-wireframe">
      <header className="flex shrink-0 items-end justify-between gap-4 border-b border-border bg-background px-8 py-6">
        <div className="min-w-0">
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            Wiki
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Shared product context — intent, architecture, and delivery notes for people and Planner.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="shrink-0"
          disabled={refreshing}
          onClick={refreshKnowledge}
        >
          {refreshing ? "Refreshing…" : "Refresh knowledge"}
        </Button>
      </header>

      <div className="flex min-h-0 flex-1">
        <aside className="flex w-[240px] shrink-0 flex-col border-r border-border bg-muted/40">
          <div className="min-h-0 flex-1 overflow-y-auto px-2 py-3">
            {documents.length === 0 ? (
              <p className="px-2 text-[13px] text-muted-foreground">No documents yet.</p>
            ) : (
              <ul className="flex flex-col gap-0.5">
                {tree.map((node) => (
                  <TreeNode
                    key={node.kind === "folder" ? `folder:${node.name}` : node.document.id}
                    node={node}
                    depth={0}
                    path={node.name}
                    selectedId={selectedId}
                    expanded={expanded}
                    onToggle={toggleFolder}
                    onSelect={setSelectedId}
                  />
                ))}
              </ul>
            )}
          </div>
        </aside>

        <section className="min-w-0 flex-1 overflow-y-auto px-8 py-6">
          {!selected ? (
            <p className="text-[13px] text-muted-foreground">Select a document to read or edit.</p>
          ) : editing ? (
            <div className="mx-auto flex w-full max-w-[720px] flex-col gap-3">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <Input
                    value={draftTitle}
                    onChange={(event) => setDraftTitle(event.target.value)}
                    aria-label="Document title"
                    className="h-9 text-[15px] font-medium"
                  />
                  <p className="mt-1 text-[12px] text-muted-foreground">{selected.path}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setDraftTitle(selected.title);
                      setDraftContent(selected.content);
                      setEditing(false);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button type="button" size="sm" onClick={saveEdit}>
                    Save
                  </Button>
                </div>
              </div>
              <Textarea
                value={draftContent}
                onChange={(event) => setDraftContent(event.target.value)}
                aria-label="Document content"
                spellCheck={false}
                className="min-h-[420px] resize-y font-mono text-[13px] leading-relaxed"
              />
            </div>
          ) : (
            <div className="mx-auto w-full max-w-[720px]">
              <div className="mb-4 flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="text-[18px] font-semibold tracking-[-0.02em] text-foreground">{selected.title}</h2>
                  <p className="mt-0.5 text-[12px] text-muted-foreground">{selected.path}</p>
                </div>
                <Button type="button" variant="outline" size="sm" onClick={() => setEditing(true)}>
                  Edit
                </Button>
              </div>
              <div className="prose-wiki text-[13px] leading-relaxed text-foreground [&_code]:rounded [&_code]:bg-accent [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[12px] [&_h1]:mb-3 [&_h1]:text-[22px] [&_h1]:font-semibold [&_h1]:tracking-[-0.02em] [&_h2]:mt-5 [&_h2]:mb-2 [&_h2]:text-[15px] [&_h2]:font-semibold [&_li]:my-0.5 [&_p]:my-2 [&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded-lg [&_pre]:border [&_pre]:border-border [&_pre]:bg-accent/40 [&_pre]:p-3 [&_pre]:text-[12px] [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5">
                <Markdown remarkPlugins={[remarkGfm]}>{selected.content}</Markdown>
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
