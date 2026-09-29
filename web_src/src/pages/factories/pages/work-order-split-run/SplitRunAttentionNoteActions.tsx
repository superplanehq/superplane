import { Bug, ExternalLink, Loader2, Play } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import type { SplitRunFooterAction } from "./splitRunFooter";
import { noteActionClassName, noteActionDisabled } from "./splitRunNoteActionStyle";

export function NoteCta({ label, href, icon }: { label: string; href: string; icon?: "bug" }) {
  const external = href.startsWith("http");
  const mark = icon === "bug" ? <Bug className="size-3.5" aria-hidden /> : null;
  return (
    <Button asChild size="sm" variant="outline">
      {external ? (
        <a href={href} target="_blank" rel="noreferrer">
          {mark}
          {label}
          <ExternalLink className="size-3.5" aria-hidden />
        </a>
      ) : (
        <Link href={href}>
          {mark}
          {label}
        </Link>
      )}
    </Button>
  );
}

function ActionIcon({ kind, strip }: { kind?: SplitRunFooterAction["kind"]; strip?: boolean }) {
  if (strip && kind === "start") {
    return <Play className="size-3.5" aria-hidden />;
  }
  return null;
}

export function NoteAction({
  action,
  actionBusy,
  startBusy,
  startDisabled,
  grouped = false,
  strip = false,
  variant,
  onClick,
}: {
  action: SplitRunFooterAction;
  actionBusy: boolean;
  startBusy: boolean;
  startDisabled: boolean;
  grouped?: boolean;
  /** On the refine strip Start carries a play icon. */
  strip?: boolean;
  /** Overrides the emphasis from the footer action. */
  variant?: "default" | "outline" | "ghost";
  onClick: () => void;
}) {
  const primary = action.emphasis === "primary";
  const busy = action.kind === "start" ? startBusy : actionBusy;
  const disabled = noteActionDisabled(action.kind, {
    actionBusy,
    startBusy,
    startDisabled,
    actionDisabled: action.disabled,
  });

  const button = (
    <Button
      type="button"
      size="sm"
      variant={variant ?? (primary ? "default" : "outline")}
      disabled={disabled}
      onClick={onClick}
      className={noteActionClassName({ grouped })}
      data-testid={primary ? "split-run-review-cta" : `split-run-footer-${action.id}`}
    >
      {busy ? (
        <Loader2 className="size-3.5 animate-spin" aria-hidden />
      ) : (
        <ActionIcon kind={action.kind} strip={strip} />
      )}
      {action.label}
    </Button>
  );
  if (grouped || !action.tooltip) {
    return button;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>{disabled ? <span className="inline-flex">{button}</span> : button}</TooltipTrigger>
      <TooltipContent>{action.tooltip}</TooltipContent>
    </Tooltip>
  );
}
