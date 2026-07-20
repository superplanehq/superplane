import { CommandGroup, CommandItem, CommandSeparator } from "@/components/ui/command";
import { appPath } from "@/lib/appPaths";
import { ActionItem, CanvasListItems } from "./items";
import { BookOpen, ChevronLeft, ChevronRight, Key, Link2, LogOut, Palette, Plug, Plus, UserPlus } from "lucide-react";
import type { CanvasCommandListProps, PaletteAction } from "./types";

export type IntegrationStatus = "ready" | "pending" | "error";

export type IntegrationItem = {
  id: string;
  name: string;
  providerName: string;
  status: IntegrationStatus;
};

export type CommandPalettePageProps = {
  canvasListProps: CanvasCommandListProps;
  integrations: IntegrationItem[];
  onCreateApp: () => void;
  onCopyInviteLink: () => void;
  showCopyInviteLink: boolean;
  copyInviteLinkDisabled: boolean;
  onExpandApps: () => void;
  onExpandIntegrations: () => void;
  onCollapse: () => void;
  onGoToDocs: () => void;
  onNewAPIKey: () => void;
  onNewSecret: () => void;
  onSignOut: () => void;
  onConnectIntegration: () => void;
  onSelectIntegration: (id: string) => void;
  expandedSection: "apps" | "integrations" | null;
  createAppLabel: string;
  createAppDisabled: boolean;
  // Search mode
  searchActive: boolean;
  searchResults: PaletteAction[];
  handleSetSearch: (value: string) => void;
  handleOpenChange: (open: boolean) => void;
};

export function CommandPalettePage(props: CommandPalettePageProps) {
  if (props.searchActive) {
    return <SearchResults results={props.searchResults} />;
  }
  return <DefaultView {...props} />;
}

function SearchResults({ results }: { results: PaletteAction[] }) {
  if (results.length === 0) return null;
  return (
    <CommandGroup>
      {results.map((action) => (
        <ActionItem key={action.id} action={action} />
      ))}
    </CommandGroup>
  );
}

function DefaultView(props: CommandPalettePageProps) {
  return (
    <>
      <CommandGroup>
        <CommandItem
          value="new-app"
          onSelect={props.onCreateApp}
          disabled={props.createAppDisabled}
          className="cursor-pointer"
        >
          <Plus className="mr-2 size-4 shrink-0" />
          <span>{props.createAppLabel}</span>
        </CommandItem>
        {props.showCopyInviteLink ? (
          <CommandItem
            value="copy-invite-link"
            onSelect={props.onCopyInviteLink}
            disabled={props.copyInviteLinkDisabled}
            className="cursor-pointer"
          >
            <Link2 className="mr-2 size-4 shrink-0" />
            <span>Copy Invite Link</span>
          </CommandItem>
        ) : null}
      </CommandGroup>

      <CommandSeparator className="my-2" />

      <CommandGroup heading="Quick Links">
        {props.expandedSection === "apps" ? (
          <>
            <CommandItem value="back-from-apps" onSelect={props.onCollapse} className="cursor-pointer">
              <ChevronLeft className="mr-2 size-4 shrink-0 text-slate-400" />
              <span className="text-slate-500">Back</span>
            </CommandItem>
            <ExpandedAppList {...props.canvasListProps} />
          </>
        ) : (
          <CommandItem value="search-apps" onSelect={props.onExpandApps} className="cursor-pointer">
            <Palette className="mr-2 size-4 shrink-0" />
            <span className="flex-1">Apps</span>
            <ChevronRight className="ml-auto size-4 shrink-0 text-slate-400" />
          </CommandItem>
        )}

        {props.expandedSection === "integrations" ? (
          <>
            <CommandItem value="back-from-integrations" onSelect={props.onCollapse} className="cursor-pointer">
              <ChevronLeft className="mr-2 size-4 shrink-0 text-slate-400" />
              <span className="text-slate-500">Back</span>
            </CommandItem>
            <ExpandedIntegrationList
              integrations={props.integrations}
              onConnectNew={props.onConnectIntegration}
              onSelectIntegration={(id) => {
                props.onSelectIntegration?.(id);
              }}
            />
          </>
        ) : (
          <CommandItem value="connected-integrations" onSelect={props.onExpandIntegrations} className="cursor-pointer">
            <Plug className="mr-2 size-4 shrink-0" />
            <span className="flex-1">Integrations</span>
            <ChevronRight className="ml-auto size-4 shrink-0 text-slate-400" />
          </CommandItem>
        )}

        <CommandItem value="go-to-docs" onSelect={props.onGoToDocs} className="cursor-pointer">
          <BookOpen className="mr-2 size-4 shrink-0" />
          <span>Go to Docs</span>
        </CommandItem>

        <CommandItem value="new-api-key" onSelect={props.onNewAPIKey} className="cursor-pointer">
          <UserPlus className="mr-2 size-4 shrink-0" />
          <span>New API Key</span>
        </CommandItem>

        <CommandItem value="new-secret" onSelect={props.onNewSecret} className="cursor-pointer">
          <Key className="mr-2 size-4 shrink-0" />
          <span>New Secret</span>
        </CommandItem>

        <CommandItem value="sign-out" onSelect={props.onSignOut} className="cursor-pointer">
          <LogOut className="mr-2 size-4 shrink-0" />
          <span>Sign Out</span>
        </CommandItem>
      </CommandGroup>
    </>
  );
}

function ExpandedAppList({ goTo, organizationId, ...props }: CanvasCommandListProps) {
  return (
    <CanvasListItems
      {...props}
      description="Open app"
      emptyLabel="No apps available."
      icon={Palette}
      onSelect={(canvas) => {
        const id = canvas.id;
        if (organizationId && id) goTo(appPath(organizationId, id));
      }}
    />
  );
}

function ExpandedIntegrationList({
  integrations,
  onConnectNew,
  onSelectIntegration,
}: {
  integrations: IntegrationItem[];
  onConnectNew: () => void;
  onSelectIntegration: (id: string) => void;
}) {
  return (
    <>
      {integrations.map((integration) => (
        <CommandItem
          key={integration.id}
          value={`integration-${integration.name}`}
          onSelect={() => onSelectIntegration(integration.id)}
          className="cursor-pointer"
        >
          <Plug className="mr-2 size-4 shrink-0" />
          <span className="flex-1">{integration.name}</span>
          <span
            className={`text-xs ${
              integration.status === "ready"
                ? "text-green-600"
                : integration.status === "error"
                  ? "text-red-500"
                  : "text-amber-600"
            }`}
          >
            {integration.status}
          </span>
        </CommandItem>
      ))}
      <CommandItem value="connect-new-integration" onSelect={onConnectNew} className="cursor-pointer">
        <Plus className="mr-2 size-4 shrink-0 text-slate-400" />
        <span className="text-slate-500">Connect new integration...</span>
      </CommandItem>
    </>
  );
}
