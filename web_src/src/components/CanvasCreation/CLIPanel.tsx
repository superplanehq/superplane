import { useState } from "react";
import { BookOpen, ExternalLink, KeyRound, Loader2 } from "lucide-react";
import { meRegenerateToken } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { CopyButton } from "./CopyButton";

const CLI_COMMANDS = [
  { label: "Create a canvas from template", command: "superplane canvases create -f canvas.yaml" },
  { label: "List available components", command: "superplane index components" },
  { label: "List available triggers", command: "superplane index triggers" },
];

function detectPlatform(): string {
  const ua = navigator.userAgent.toLowerCase();
  const isLinux = ua.includes("linux");
  const isArm = ua.includes("arm") || ua.includes("aarch64");
  const os = isLinux ? "linux" : "darwin";
  const arch = isArm ? "arm64" : "amd64";
  return `${os}-${arch}`;
}

export function CLIPanel({ organizationId }: { organizationId: string }) {
  const platform = detectPlatform();
  const installCommand = `curl -L https://install.superplane.com/superplane-cli-${platform} -o superplane && chmod +x superplane && sudo mv superplane /usr/local/bin/`;
  const [connectCommand, setConnectCommand] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);

  const handleGenerateConnect = async () => {
    try {
      setGenerating(true);
      const response = await meRegenerateToken(withOrganizationHeader({ organizationId }));
      const token = response.data?.token;
      if (!token) {
        showErrorToast("Failed to generate API token");
        return;
      }
      const baseURL = window.location.origin;
      const cmd = `superplane connect ${baseURL} ${token}`;
      setConnectCommand(cmd);
      await navigator.clipboard.writeText(cmd);
      showSuccessToast("Connect command copied to clipboard");
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : "Failed to generate token");
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="bg-gray-900 rounded-xl p-4 font-mono text-sm">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">
            Install ({platform})
          </span>
          <CopyButton text={installCommand} />
        </div>
        <div className="text-green-400 break-all leading-relaxed">
          <span className="text-gray-500 select-none">$ </span>
          {installCommand}
        </div>
        <a
          href="https://docs.superplane.com/installation/cli"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block mt-2 text-[11px] font-sans text-gray-500 hover:text-gray-300 transition-colors"
        >
          Other platforms
        </a>
      </div>

      <div className="bg-gray-900 rounded-xl p-4 font-mono text-sm">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">Connect</span>
          {connectCommand && <CopyButton text={connectCommand} />}
        </div>
        {connectCommand ? (
          <div className="text-gray-300 break-all">
            <span className="text-gray-500 select-none">$ </span>
            {connectCommand}
          </div>
        ) : (
          <div>
            <div className="text-[11px] font-sans text-gray-500 mb-2.5">
              Generate a personal API token and get a ready-to-paste connect command.
            </div>
            <button
              type="button"
              onClick={handleGenerateConnect}
              disabled={generating}
              className="inline-flex items-center gap-1.5 font-sans text-[12px] font-medium text-gray-900 bg-gray-100 hover:bg-white px-3 py-1.5 rounded-md transition-colors disabled:opacity-50"
            >
              {generating ? <Loader2 size={12} className="animate-spin" /> : <KeyRound size={12} />}
              {generating ? "Generating..." : "Generate connect command"}
            </button>
          </div>
        )}
        <div className="mt-2">
          <a
            href={`/${organizationId}/settings/profile`}
            className="text-[11px] font-sans text-gray-500 hover:text-gray-300 transition-colors"
          >
            Manage your API token in Settings
          </a>
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-4 font-mono text-sm">
        <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">
          Quick reference
        </span>
        <div className="mt-3 space-y-3">
          {CLI_COMMANDS.map((cmd) => (
            <div key={cmd.command}>
              <div className="text-[11px] font-sans text-gray-500 mb-0.5">{cmd.label}</div>
              <div className="flex items-center justify-between gap-2">
                <div className="text-gray-300 truncate">
                  <span className="text-gray-500 select-none">$ </span>
                  {cmd.command}
                </div>
                <CopyButton text={cmd.command} />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-4 font-mono text-sm">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">AI Skills</span>
          <CopyButton text="npx skills add superplanehq/skills" />
        </div>
        <div className="text-[11px] font-sans text-gray-500 mb-1.5">
          Install skills for AI agents (Cursor, Claude Code, Codex, etc.)
        </div>
        <div className="text-gray-300">
          <span className="text-gray-500 select-none">$ </span>
          npx skills add superplanehq/skills
        </div>
      </div>

      <div className="flex items-center gap-4 mt-4">
        <a
          href="https://docs.superplane.com/installation/cli"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1.5 text-[12px] text-gray-500 dark:text-gray-400 hover:text-primary transition-colors"
        >
          <BookOpen size={13} />
          CLI docs
          <ExternalLink size={10} />
        </a>
        <a
          href="https://docs.superplane.com/get-started/quickstart"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1.5 text-[12px] text-gray-500 dark:text-gray-400 hover:text-primary transition-colors"
        >
          <BookOpen size={13} />
          Quickstart guide
          <ExternalLink size={10} />
        </a>
      </div>
    </div>
  );
}
