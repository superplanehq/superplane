import { useState } from "react";
import { BookOpen, ChevronDown, ExternalLink, KeyRound, Loader2 } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { detectPlatform, getInstallCommand, useConnectCommand } from "@/utils/cli";
import { CopyButton } from "@/ui/CopyButton";

function CommandRow({ label, command }: { label: string; command: string }) {
  return (
    <div>
      <div className="text-[11px] font-sans text-gray-500 mb-0.5">{label}</div>
      <div className="flex items-center justify-between gap-2">
        <div className="text-gray-300 truncate">
          <span className="text-gray-500 select-none">$ </span>
          {command}
        </div>
        <CopyButton text={command} dark />
      </div>
    </div>
  );
}

function InstallConnectSection({
  organizationId,
  connectCommand,
  generating,
  onGenerateConnect,
}: {
  organizationId?: string;
  connectCommand: string | null;
  generating: boolean;
  onGenerateConnect: () => void;
}) {
  const [isInstallOpen, setIsInstallOpen] = useState(false);

  const platform = detectPlatform();
  const installCommand = getInstallCommand(platform);

  return (
    <div className="border-t border-gray-200 dark:border-gray-700">
      <Collapsible open={isInstallOpen} onOpenChange={setIsInstallOpen}>
        <CollapsibleTrigger className="flex w-full items-center gap-1.5 px-4 py-2.5 text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors">
          <ChevronDown size={12} className={`transition-transform ${isInstallOpen ? "" : "-rotate-90"}`} />
          How to install SuperPlane CLI and connect
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="px-4 pb-3 space-y-3">
            <div className="bg-gray-900 rounded-lg p-3 font-mono text-sm">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">
                  1. Install ({platform})
                </span>
                <CopyButton text={installCommand} dark />
              </div>
              <div className="text-green-400 break-all leading-relaxed text-xs">
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

            <div className="bg-gray-900 rounded-lg p-3 font-mono text-sm">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">
                  2. Connect to the organization
                </span>
                {connectCommand && <CopyButton text={connectCommand} dark />}
              </div>
              {connectCommand ? (
                <div className="text-gray-300 break-all text-xs">
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
                    onClick={onGenerateConnect}
                    disabled={generating || !organizationId}
                    className="inline-flex items-center gap-1.5 font-sans text-[12px] font-medium text-gray-900 bg-gray-100 hover:bg-white px-3 py-1.5 rounded-md transition-colors disabled:opacity-50"
                  >
                    {generating ? <Loader2 size={12} className="animate-spin" /> : <KeyRound size={12} />}
                    {generating ? "Generating..." : "Generate connect command"}
                  </button>
                </div>
              )}
              {organizationId && (
                <div className="mt-2">
                  <a
                    href={`/${organizationId}/settings/profile`}
                    className="text-[11px] font-sans text-gray-500 hover:text-gray-300 transition-colors"
                  >
                    Manage your API token in Settings
                  </a>
                </div>
              )}
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}

interface CliCommandsPopoverProps {
  canvasId?: string;
  organizationId?: string;
}

export function CliCommandsPopover({ canvasId, organizationId }: CliCommandsPopoverProps) {
  const { connectCommand, generating, handleGenerateConnect } = useConnectCommand(organizationId);
  const commands: { label: string; command: string }[] = [];

  if (canvasId) {
    commands.push({
      label: "Describe canvas",
      command: `superplane canvases get ${canvasId}`,
    });
    commands.push({
      label: "List events",
      command: `superplane events list --canvas-id ${canvasId}`,
    });
    commands.push({
      label: "Update canvas from file",
      command: `superplane canvases update -f canvas.yaml --canvas-id ${canvasId}`,
    });
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button type="button" className="rounded-sm px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100">
          CLI
        </button>
      </PopoverTrigger>

      <PopoverContent align="center" className="w-[26rem] p-0" sideOffset={8}>
        <div className="px-4 pt-3 pb-2">
          {commands.length > 0 ? (
            <div className="bg-gray-900 rounded-lg p-3 font-mono text-sm">
              <span className="text-[11px] font-sans font-medium text-gray-400 uppercase tracking-wider">
                Canvas commands
              </span>
              <div className="mt-2.5 space-y-2.5">
                {commands.map((cmd) => (
                  <CommandRow key={cmd.command} label={cmd.label} command={cmd.command} />
                ))}
              </div>
            </div>
          ) : (
            <div className="bg-gray-900 rounded-lg p-3 text-sm">
              <p className="text-gray-500 font-sans">Save the canvas to see contextual CLI commands.</p>
            </div>
          )}
        </div>

        <InstallConnectSection
          organizationId={organizationId}
          connectCommand={connectCommand}
          generating={generating}
          onGenerateConnect={handleGenerateConnect}
        />

        <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-2.5">
          <a
            href="https://docs.superplane.com/installation/cli"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400 hover:text-primary transition-colors"
          >
            <BookOpen size={12} />
            CLI reference
            <ExternalLink size={10} />
          </a>
        </div>
      </PopoverContent>
    </Popover>
  );
}
