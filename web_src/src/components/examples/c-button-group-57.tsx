import { useState } from "react";
import { ChevronDown, MessageSquare, ShieldCheck, Zap } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ButtonGroup, ButtonGroupSeparator } from "@/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

const modes = [
  { id: "assist", label: "Assist", Icon: MessageSquare },
  { id: "review", label: "Review", Icon: ShieldCheck },
  { id: "auto", label: "Auto", Icon: Zap },
] as const;

const limits = ["2k credits", "10k credits", "Unlimited"];

type ModeId = (typeof modes)[number]["id"];

/** ReUI c-button-group-57: agent mode selector with a credit-cap dropdown. */
export function Pattern() {
  const [active, setActive] = useState<ModeId>("review");
  const [limit, setLimit] = useState(limits[1]);

  return (
    <ButtonGroup>
      {modes.map((mode) => (
        <Button
          key={mode.id}
          variant="outline"
          size="sm"
          className={cn(active === mode.id && "bg-muted")}
          onClick={() => setActive(mode.id)}
        >
          <mode.Icon className="size-3.5 opacity-60" aria-hidden="true" />
          {mode.label}
        </Button>
      ))}
      <ButtonGroupSeparator />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1.5 text-xs" aria-label="Select credit cap">
            {limit}
            <ChevronDown className="size-3 opacity-60" aria-hidden="true" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-36">
          <DropdownMenuGroup>
            {limits.map((item) => (
              <DropdownMenuItem key={item} onClick={() => setLimit(item)}>
                {item}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </ButtonGroup>
  );
}
