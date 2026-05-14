import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Loader2, Pencil, Save, X } from "lucide-react";
import { Editor } from "@monaco-editor/react";
import { CanvasMarkdown, type NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";

interface FileViewerProps {
  canvasId: string;
  path: string;
  nodeRefs?: NodeChipContext;
  onSaved?: () => void;
}

export function FileViewer({ canvasId, path, nodeRefs, onSaved }: FileViewerProps) {
  const [content, setContent] = useState("");
  const [editContent, setEditContent] = useState("");
  const [editable, setEditable] = useState(false);
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchFile = useCallback(async () => {
    setLoading(true);
    setEditing(false);
    setError(null);
    try {
      const res = await fetch(`/api/repo/${canvasId}/files/${path}`, {
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Failed to load: ${res.status}`);
      const data = await res.json();
      setContent(data.content);
      setEditContent(data.content);
      setEditable(data.editable);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [canvasId, path]);

  useEffect(() => {
    fetchFile();
  }, [fetchFile]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      const res = await fetch(`/api/repo/${canvasId}/files/${path}`, {
        method: "PUT",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: editContent }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `Save failed: ${res.status}`);
      }
      setContent(editContent);
      setEditing(false);
      onSaved?.();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }, [canvasId, path, editContent, onSaved]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-red-500">{error}</div>
    );
  }

  const isYaml = path.endsWith(".yaml") || path.endsWith(".yml");
  const isJson = path.endsWith(".json");
  const isMd = path.endsWith(".md");

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex h-10 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-slate-700">{path}</span>
          {!editable && (
            <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-500">read-only</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {editable && !editing && (
            <Button size="sm" variant="outline" onClick={() => { setEditContent(content); setEditing(true); }}>
              <Pencil className="mr-1 h-3.5 w-3.5" />
              Edit
            </Button>
          )}
          {editing && (
            <>
              <Button size="sm" variant="ghost" onClick={() => setEditing(false)}>
                <X className="mr-1 h-3.5 w-3.5" />
                Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={saving || editContent === content}>
                {saving ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1 h-3.5 w-3.5" />}
                Save
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {editing ? (
          <Editor
            height="100%"
            language="markdown"
            value={editContent}
            onChange={(val) => setEditContent(val || "")}
            theme="vs"
            options={{
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              wordWrap: "on",
              scrollBeyondLastLine: false,
              tabSize: 2,
              renderLineHighlight: "line",
              automaticLayout: true,
            }}
          />
        ) : isMd ? (
          <div className="h-full overflow-auto p-6 prose prose-sm prose-slate max-w-none">
            <CanvasMarkdown canvasId={canvasId} nodeRefs={nodeRefs}>{content}</CanvasMarkdown>
          </div>
        ) : (
          <Editor
            height="100%"
            language={isYaml ? "yaml" : isJson ? "json" : "plaintext"}
            value={content}
            theme="vs"
            options={{
              readOnly: true,
              domReadOnly: true,
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              wordWrap: "on",
              folding: true,
              scrollBeyondLastLine: false,
              tabSize: 2,
              renderLineHighlight: "line",
              automaticLayout: true,
            }}
          />
        )}
      </div>
    </div>
  );
}
