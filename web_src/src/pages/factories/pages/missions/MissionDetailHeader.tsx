import { Button } from "@/components/ui/button";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Check, Ellipsis, Link2 } from "lucide-react";
import { useState } from "react";

import { type MissionCloseReason, type MissionDisplayStatus } from "./missionListModel";

interface MissionDetailHeaderProps {
  missionName: string;
  status: MissionDisplayStatus;
  isManuallyClosed: boolean;
  onClose: (reason: MissionCloseReason) => void;
  onReopen: () => void;
}

export function MissionDetailHeader({
  missionName,
  status,
  isManuallyClosed,
  onClose,
  onReopen,
}: MissionDetailHeaderProps) {
  return (
    <header className="sticky top-0 z-10 col-span-full mx-[calc(var(--workspace-page-gutter)*-1)] mb-3 bg-background px-[var(--workspace-page-gutter)] py-3">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <h1 className="workspace-page-title min-w-0 flex-1">{missionName}</h1>
        <div className="flex flex-wrap items-center gap-1">
          <CopyLinkButton />
          <HeaderOverflowMenu
            status={status}
            isManuallyClosed={isManuallyClosed}
            onClose={onClose}
            onReopen={onReopen}
          />
        </div>
      </div>
    </header>
  );
}

function CopyLinkButton() {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      showSuccessToast("Link copied to clipboard.");
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      showErrorToast("Failed to copy link.");
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      onClick={() => void handleCopy()}
      className="size-7 text-muted-foreground hover:bg-accent hover:text-foreground"
      aria-label="Copy link to mission"
      data-testid="mission-copy-link-button"
    >
      {copied ? <Check className="h-4 w-4" aria-hidden /> : <Link2 className="h-4 w-4" aria-hidden />}
    </Button>
  );
}

function HeaderOverflowMenu({
  status,
  isManuallyClosed,
  onClose,
  onReopen,
}: {
  status: MissionDisplayStatus;
  isManuallyClosed: boolean;
  onClose: (reason: MissionCloseReason) => void;
  onReopen: () => void;
}) {
  if (status === "completed" && !isManuallyClosed) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-muted-foreground hover:bg-accent hover:text-foreground"
          aria-label="More actions"
          data-testid="mission-actions-button"
        >
          <Ellipsis className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {isManuallyClosed ? (
          <DropdownMenuItem
            onSelect={() => {
              onReopen();
              showSuccessToast("Mission reopened.");
            }}
            data-testid="mission-reopen-button"
          >
            Reopen
          </DropdownMenuItem>
        ) : (
          <>
            <DropdownMenuItem
              onSelect={() => {
                onClose("completed");
                showSuccessToast("Mission completed.");
              }}
              data-testid="mission-complete-button"
            >
              Complete
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={() => {
                onClose("closed");
                showSuccessToast("Mission closed.");
              }}
              data-testid="mission-close-button"
            >
              Close
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
