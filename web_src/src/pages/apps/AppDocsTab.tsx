import { useCallback, useState } from "react";
import { Editor } from "@monaco-editor/react";
import { useAppDocs, useAppDoc, useUpdateAppDoc } from "@/hooks/useAppData";
import { WorkflowMarkdownPreview } from "@/pages/workflowv2/WorkflowMarkdownPreview";
import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ChevronRight, FileText, Loader2, Save } from "lucide-react";
import { cn } from "@/lib/utils";

interface AppDocsTabProps {
  appId: string;
  readOnly?: boolean;
}

export function AppDocsTab({ appId, readOnly = false }: AppDocsTabProps) {
  const [selectedPath, setSelectedPath] = useState<string>("docs/README.md");
  const [editContent, setEditContent] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  const docsQuery = useAppDocs(appId);
  const docQuery = useAppDoc(appId, selectedPath, !!selectedPath);
  const updateDocMutation = useUpdateAppDoc(appId);

  const docs = docsQuery.data ?? [];

  const currentContent = editContent !== null ? editContent : (docQuery.data?.content ?? "");

  const handleSelectPath = useCallback((path: string) => {
    setSelectedPath(path);
    setEditContent(null);
    setIsEditing(false);
  }, []);

  const handleStartEdit = () => {
    setEditContent(docQuery.data?.content ?? "");
    setIsEditing(true);
  };

  const handleCancelEdit = () => {
    setEditContent(null);
    setIsEditing(false);
  };

  const handleSave = async () => {
    if (editContent === null) return;
    try {
      await updateDocMutation.mutateAsync({ path: selectedPath, content: editContent });
      showSuccessToast("Doc saved and committed");
      setIsEditing(false);
      setEditContent(null);
    } catch {
      showErrorToast("Failed to save doc");
    }
  };

  return (
    <div className="flex h-full overflow-hidden">
      {/* File tree sidebar */}
      <aside className="w-56 shrink-0 border-r border-slate-200 dark:border-slate-700 overflow-y-auto bg-white dark:bg-slate-900">
        <div className="px-3 py-2 border-b border-slate-200 dark:border-slate-700">
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Docs</span>
        </div>
        {docsQuery.isLoading ? (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : docs.length === 0 ? (
          <div className="px-3 py-4 text-sm text-muted-foreground">No documents yet.</div>
        ) : (
          <ul className="py-1">
            {docs.map((doc) => (
              <li key={doc.path}>
                <button
                  onClick={() => handleSelectPath(doc.path ?? "")}
                  className={cn(
                    "flex items-center gap-1.5 w-full text-left px-3 py-1.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors",
                    selectedPath === doc.path && "bg-slate-100 dark:bg-slate-800 font-medium",
                  )}
                >
                  <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <span className="truncate font-mono text-xs">{doc.path?.replace("docs/", "") ?? ""}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </aside>

      {/* Doc content area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {selectedPath ? (
          <>
            {/* Doc toolbar */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
              <div className="flex items-center gap-1 text-sm text-muted-foreground">
                <ChevronRight className="h-3.5 w-3.5" />
                <span className="font-mono text-xs">{selectedPath.replace("docs/", "")}</span>
              </div>
              {!readOnly && (
                <div className="flex items-center gap-2">
                  {isEditing ? (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleCancelEdit}
                        disabled={updateDocMutation.isPending}
                      >
                        Cancel
                      </Button>
                      <Button size="sm" onClick={handleSave} disabled={updateDocMutation.isPending}>
                        {updateDocMutation.isPending ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" />
                        ) : (
                          <Save className="h-3.5 w-3.5 mr-1" />
                        )}
                        Save
                      </Button>
                    </>
                  ) : (
                    <Button variant="outline" size="sm" onClick={handleStartEdit}>
                      Edit
                    </Button>
                  )}
                </div>
              )}
            </div>

            {/* Content */}
            <div className="flex-1 overflow-hidden">
              {docQuery.isLoading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : isEditing ? (
                <Editor
                  height="100%"
                  defaultLanguage="markdown"
                  value={currentContent}
                  onChange={(val) => setEditContent(val ?? "")}
                  options={{
                    minimap: { enabled: false },
                    lineNumbers: "on",
                    wordWrap: "on",
                    scrollBeyondLastLine: false,
                    fontSize: 13,
                    fontFamily: "monospace",
                  }}
                />
              ) : (
                <div className="h-full overflow-auto px-8 py-6 bg-white dark:bg-slate-900">
                  {currentContent ? (
                    <WorkflowMarkdownPreview content={currentContent} className="max-w-3xl" />
                  ) : (
                    <p className="text-sm text-muted-foreground italic">This document is empty.</p>
                  )}
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
            Select a document to view
          </div>
        )}
      </div>
    </div>
  );
}
