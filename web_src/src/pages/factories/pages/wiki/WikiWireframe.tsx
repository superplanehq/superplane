import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ChevronRight, FileText, Folder } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { factoryPageSubtitleClassName, factoryPageTitleClassName } from "../factoryPageLayoutStyles";
import { buildWikiTree, wikiFolderPaths, type WikiDocument, type WikiTreeNode } from "./wikiMocks";

export type WikiWireframeProps = {
  /** Documents shown when the wireframe mounts. Supplied by stories — no baked-in fixtures. */
  initialDocuments?: WikiDocument[];
  /** Corpus swapped in by “Refresh knowledge”. Supplied by stories. */
  refreshedDocuments?: WikiDocument[];
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

function WikiDocumentEditor({
  document,
  draftTitle,
  draftContent,
  onDraftTitleChange,
  onDraftContentChange,
  onCancel,
  onSave,
}: {
  document: WikiDocument;
  draftTitle: string;
  draftContent: string;
  onDraftTitleChange: (value: string) => void;
  onDraftContentChange: (value: string) => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  return (
    <div className="mx-auto flex w-full max-w-[720px] flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <Input
            value={draftTitle}
            onChange={(event) => onDraftTitleChange(event.target.value)}
            aria-label="Document title"
            className="h-9 text-[15px] font-medium"
          />
          <p className="mt-1 text-[12px] text-muted-foreground">{document.path}</p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="button" size="sm" onClick={onSave}>
            Save
          </Button>
        </div>
      </div>
      <Textarea
        value={draftContent}
        onChange={(event) => onDraftContentChange(event.target.value)}
        aria-label="Document content"
        spellCheck={false}
        className="min-h-[420px] resize-y font-mono text-[13px] leading-relaxed"
      />
    </div>
  );
}

function WikiDocumentReader({ document, onEdit }: { document: WikiDocument; onEdit: () => void }) {
  return (
    <div className="mx-auto w-full max-w-[720px]">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-[18px] font-semibold tracking-[-0.02em] text-foreground">{document.title}</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">{document.path}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={onEdit}>
          Edit
        </Button>
      </div>
      <div className="prose-wiki text-[13px] leading-relaxed text-foreground [&_code]:rounded [&_code]:bg-accent [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[12px] [&_h1]:mb-3 [&_h1]:text-[22px] [&_h1]:font-semibold [&_h1]:tracking-[-0.02em] [&_h2]:mt-5 [&_h2]:mb-2 [&_h2]:text-[15px] [&_h2]:font-semibold [&_li]:my-0.5 [&_p]:my-2 [&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded-lg [&_pre]:border [&_pre]:border-border [&_pre]:bg-accent/40 [&_pre]:p-3 [&_pre]:text-[12px] [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5">
        <Markdown remarkPlugins={[remarkGfm]}>{document.content}</Markdown>
      </div>
    </div>
  );
}

function WikiDocumentPane({
  selected,
  editing,
  draftTitle,
  draftContent,
  onDraftTitleChange,
  onDraftContentChange,
  onCancelEdit,
  onSaveEdit,
  onStartEdit,
}: {
  selected: WikiDocument | null;
  editing: boolean;
  draftTitle: string;
  draftContent: string;
  onDraftTitleChange: (value: string) => void;
  onDraftContentChange: (value: string) => void;
  onCancelEdit: () => void;
  onSaveEdit: () => void;
  onStartEdit: () => void;
}) {
  if (!selected) {
    return <p className="text-[13px] text-muted-foreground">Select a document to read or edit.</p>;
  }

  if (editing) {
    return (
      <WikiDocumentEditor
        document={selected}
        draftTitle={draftTitle}
        draftContent={draftContent}
        onDraftTitleChange={onDraftTitleChange}
        onDraftContentChange={onDraftContentChange}
        onCancel={onCancelEdit}
        onSave={onSaveEdit}
      />
    );
  }

  return <WikiDocumentReader document={selected} onEdit={onStartEdit} />;
}

function WikiWireframeHeader({ refreshing, onRefresh }: { refreshing: boolean; onRefresh: () => void }) {
  return (
    <header className="flex shrink-0 items-end justify-between gap-4 border-b border-border bg-background px-8 py-6">
      <div className="min-w-0">
        <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
          Wiki
        </Heading>
        <p className={cn("mt-1", factoryPageSubtitleClassName)}>
          Shared product context — intent, architecture, and delivery notes for people and Planner.
        </p>
      </div>
      <Button type="button" variant="outline" size="sm" className="shrink-0" disabled={refreshing} onClick={onRefresh}>
        {refreshing ? "Refreshing…" : "Refresh knowledge"}
      </Button>
    </header>
  );
}

function WikiTreeAside({
  documents,
  tree,
  selectedId,
  expanded,
  onToggle,
  onSelect,
}: {
  documents: WikiDocument[];
  tree: WikiTreeNode[];
  selectedId: string | null;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  onSelect: (id: string) => void;
}) {
  return (
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
                onToggle={onToggle}
                onSelect={onSelect}
              />
            ))}
          </ul>
        )}
      </div>
    </aside>
  );
}

/**
 * Storybook-only wiki wireframe (v3 parity). Not mounted on the app `/wiki` route.
 * Mock corpora must be passed from stories — this component holds no sample documents.
 */
export function WikiWireframe({
  initialDocuments = [],
  refreshedDocuments = [],
  startEditing = false,
}: WikiWireframeProps) {
  const initialSelected = initialDocuments[0] ?? null;
  const [documents, setDocuments] = useState<WikiDocument[]>(initialDocuments);
  const [selectedId, setSelectedId] = useState<string | null>(initialSelected?.id ?? null);
  const [expanded, setExpanded] = useState<Set<string>>(
    () => new Set(wikiFolderPaths(buildWikiTree(initialDocuments))),
  );
  const [editing, setEditing] = useState(startEditing);
  const [draftTitle, setDraftTitle] = useState(initialSelected?.title ?? "");
  const [draftContent, setDraftContent] = useState(initialSelected?.content ?? "");
  const [refreshing, setRefreshing] = useState(false);
  const refreshTimeoutRef = useRef<number | null>(null);

  const tree = useMemo(() => buildWikiTree(documents), [documents]);
  const selected = documents.find((doc) => doc.id === selectedId) ?? null;

  useEffect(() => {
    return () => {
      if (refreshTimeoutRef.current !== null) {
        window.clearTimeout(refreshTimeoutRef.current);
      }
    };
  }, []);

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

  return (
    <div className="flex h-screen min-h-0 flex-col" data-testid="wiki-wireframe">
      <WikiWireframeHeader
        refreshing={refreshing}
        onRefresh={() => {
          if (refreshing) return;
          setRefreshing(true);
          refreshTimeoutRef.current = window.setTimeout(() => {
            refreshTimeoutRef.current = null;
            const nextTree = buildWikiTree(refreshedDocuments);
            setDocuments(refreshedDocuments);
            setSelectedId(refreshedDocuments[0]?.id ?? null);
            setExpanded(new Set(wikiFolderPaths(nextTree)));
            setEditing(false);
            setRefreshing(false);
          }, 800);
        }}
      />

      <div className="flex min-h-0 flex-1">
        <WikiTreeAside
          documents={documents}
          tree={tree}
          selectedId={selectedId}
          expanded={expanded}
          onToggle={(path) => {
            setExpanded((current) => {
              const next = new Set(current);
              if (next.has(path)) next.delete(path);
              else next.add(path);
              return next;
            });
          }}
          onSelect={setSelectedId}
        />

        <section className="min-w-0 flex-1 overflow-y-auto px-8 py-6">
          <WikiDocumentPane
            selected={selected}
            editing={editing}
            draftTitle={draftTitle}
            draftContent={draftContent}
            onDraftTitleChange={setDraftTitle}
            onDraftContentChange={setDraftContent}
            onCancelEdit={() => {
              if (selected) {
                setDraftTitle(selected.title);
                setDraftContent(selected.content);
              }
              setEditing(false);
            }}
            onSaveEdit={() => {
              if (!selected) return;
              setDocuments((current) =>
                current.map((doc) =>
                  doc.id === selected.id
                    ? { ...doc, title: draftTitle.trim() || doc.title, content: draftContent }
                    : doc,
                ),
              );
              setEditing(false);
            }}
            onStartEdit={() => setEditing(true)}
          />
        </section>
      </div>
    </div>
  );
}
