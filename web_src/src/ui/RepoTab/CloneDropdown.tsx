import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Check, ChevronDown, Copy, Download, GitBranch } from "lucide-react";

interface CloneDropdownProps {
  repoUrl: string;
  organizationId: string;
  canvasId: string;
}

export function CloneDropdown({ repoUrl, organizationId, canvasId }: CloneDropdownProps) {
  const [copied, setCopied] = useState(false);
  const [open, setOpen] = useState(false);

  const cloneCmd = `git clone https://x-token:<your-token>@${new URL(repoUrl).host}${new URL(repoUrl).pathname}`;

  const copyToClipboard = useCallback(() => {
    navigator.clipboard.writeText(cloneCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [cloneCmd]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button size="sm" variant="default" data-testid="repo-clone-button">
          <GitBranch className="mr-1.5 h-3.5 w-3.5" />
          Clone
          <ChevronDown className="ml-1 h-3 w-3" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-0">
        <div className="p-3">
          <p className="mb-2 text-xs font-medium text-gray-700">Clone with HTTPS</p>
          <div className="flex items-center gap-1.5">
            <code className="flex-1 truncate rounded bg-gray-100 px-2.5 py-1.5 font-mono text-xs text-gray-700">
              {cloneCmd}
            </code>
            <button
              type="button"
              onClick={copyToClipboard}
              className="shrink-0 rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              title="Copy"
            >
              {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
            </button>
          </div>
        </div>

        <div className="border-t border-gray-100 p-3">
          <a
            href={`/api/repo/${canvasId}/archive`}
            download
            className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
          >
            <Download className="h-3.5 w-3.5 text-gray-400" />
            Download ZIP
          </a>
        </div>

        <div className="border-t border-gray-100 px-3 py-2.5">
          <p className="text-[11px] text-gray-500">
            Use your <strong>API token</strong> for authentication.
          </p>
          <a href={`/${organizationId}/settings/profile`} className="mt-1 block text-[11px] text-sky-600 hover:underline">
            Manage tokens →
          </a>
        </div>
      </PopoverContent>
    </Popover>
  );
}
