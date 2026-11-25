import { useContext } from "react";
import { ChevronDown, ListTree, MessageSquare } from "lucide-react";
import { Badge } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import type { SplitRunPhase } from "./splitRunMocks";
import { AutomationChatContext } from "./useAutomationChatSessions";

export function AutomationChatNavigation() {
  const chats = useContext(AutomationChatContext);
  if (!chats?.chats.length) return null;
  return (
    <ButtonGroup aria-label="Task view">
      <ChatPicker phases={chats.chats} selected={chats.choice} onSelect={chats.setChoice} navigation />
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-pressed={chats.choice === "activity"}
        className="aria-pressed:bg-accent aria-pressed:text-accent-foreground"
        onClick={() => chats.setChoice("activity")}
      >
        <ListTree className="size-4" aria-hidden />
        Timeline
      </Button>
    </ButtonGroup>
  );
}

/** A timeline card can group multiple runs, each with its own conversation. */
export function AutomationTimelineChat({ phaseIds }: { phaseIds: string[] }) {
  const chats = useContext(AutomationChatContext);
  const phases = chats?.chats.filter((phase) => phaseIds.includes(phase.id)) ?? [];
  if (!chats || !phases.length) return null;
  return (
    <span className="relative z-20">
      <ChatPicker phases={phases} selected={chats.choice} onSelect={chats.setChoice} />
    </span>
  );
}

function ChatPicker({
  phases,
  selected,
  onSelect,
  navigation = false,
}: {
  navigation?: boolean;
  phases: SplitRunPhase[];
  selected: string;
  onSelect: (runId: string) => void;
}) {
  const multiple = phases.length > 1;
  const active = phases.find((phase) => phase.runId === selected);
  const label = navigation ? (active?.componentName ?? "Chat") : "Chat";
  const button = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      aria-label={
        navigation
          ? `${label} (${phases.length} active ${phases.length === 1 ? "chat" : "chats"})`
          : `Chat with ${phases[0].componentName}`
      }
      aria-pressed={Boolean(active)}
      className="relative gap-2 aria-pressed:bg-accent aria-pressed:text-accent-foreground"
      onClick={multiple ? undefined : () => onSelect(phases[0].runId!)}
    >
      <MessageSquare className="size-4" aria-hidden />
      {label}
      {multiple ? <ChevronDown className="size-3" aria-hidden /> : null}
      {navigation ? (
        <Badge
          variant="destructive"
          size="sm"
          className="absolute -top-1.5 -right-2 z-10 rounded-full px-1"
          aria-hidden
        >
          {phases.length}
        </Badge>
      ) : null}
    </Button>
  );
  const trigger = (
    <Tooltip>
      <TooltipTrigger asChild>
        {multiple ? <DropdownMenuTrigger asChild>{button}</DropdownMenuTrigger> : button}
      </TooltipTrigger>
      <TooltipContent side="bottom">
        {phases.map((phase) => (
          <div key={phase.runId}>{phase.componentName}</div>
        ))}
      </TooltipContent>
    </Tooltip>
  );
  if (!multiple) return trigger;
  return (
    <DropdownMenu>
      {trigger}
      <DropdownMenuContent align="end" aria-label="Automation chats">
        <DropdownMenuRadioGroup value={selected} onValueChange={onSelect}>
          {phases.map((phase, index) => (
            <DropdownMenuRadioItem key={phase.runId} value={phase.runId!}>
              {phase.componentName}
              {phases.filter((entry) => entry.componentName === phase.componentName).length > 1
                ? ` · ${phases.length - index}`
                : ""}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
