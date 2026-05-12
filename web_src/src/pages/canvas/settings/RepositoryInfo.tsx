import { Copy, Check, GitBranch } from "lucide-react";
import { useState, useCallback } from "react";

interface RepositoryInfoProps {
  canvasId: string;
  canvasName: string;
  baseUrl: string;
}

function toSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "");
}

export function RepositoryInfo({ canvasId, canvasName, baseUrl }: RepositoryInfoProps) {
  const slug = toSlug(canvasName);
  const repoUrl = `${baseUrl}/git/${slug}`;
  const cloneCommand = `git clone ${repoUrl}`;
  const [copied, setCopied] = useState<string | null>(null);

  const copyToClipboard = useCallback((text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  }, []);

  return (
    <fieldset className="rounded-lg border border-slate-200 p-4">
      <legend className="flex items-center gap-2 px-1 text-sm font-medium text-slate-700">
        <GitBranch className="h-4 w-4" />
        Repository
      </legend>

      <p className="mb-3 text-sm text-slate-500">
        This canvas is backed by a git repository. Push changes to update the canvas, or edit in the UI and changes auto-commit.
      </p>

      <div className="space-y-3">
        {/* Clone URL */}
        <div>
          <label className="mb-1 block text-xs font-medium text-slate-500">Clone URL</label>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-slate-100 px-3 py-2 font-mono text-sm text-slate-800">
              {repoUrl}
            </code>
            <button
              type="button"
              onClick={() => copyToClipboard(repoUrl, "url")}
              className="rounded p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
              title="Copy URL"
            >
              {copied === "url" ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
        </div>

        {/* Clone command */}
        <div>
          <label className="mb-1 block text-xs font-medium text-slate-500">Clone command</label>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-slate-100 px-3 py-2 font-mono text-sm text-slate-800">
              {cloneCommand}
            </code>
            <button
              type="button"
              onClick={() => copyToClipboard(cloneCommand, "clone")}
              className="rounded p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
              title="Copy command"
            >
              {copied === "clone" ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
        </div>

        {/* Auth info */}
        <div className="rounded border border-slate-100 bg-slate-50 p-3">
          <p className="text-xs font-medium text-slate-600">Authentication</p>
          <p className="mt-1 text-xs text-slate-500">
            Use any username and your <strong>API token</strong> as the password.
            Generate a token from your profile settings.
          </p>
          <code className="mt-2 block rounded bg-white px-2 py-1 font-mono text-xs text-slate-600">
            git clone https://&lt;username&gt;:&lt;api-token&gt;@{new URL(repoUrl).host}/git/{slug}
          </code>
        </div>

        {/* Repo structure */}
        <div className="rounded border border-slate-100 bg-slate-50 p-3">
          <p className="text-xs font-medium text-slate-600">Repository structure</p>
          <div className="mt-2 font-mono text-xs text-slate-500">
            <div>├── canvas.yaml &nbsp;&nbsp;&nbsp;# Canvas nodes, edges, configuration</div>
            <div>├── README.md &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;# Canvas documentation</div>
            <div>├── apps/ &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;# Dashboard panels</div>
            <div>└── .superplane.yaml # Canvas/org mapping</div>
          </div>
        </div>
      </div>
    </fieldset>
  );
}
