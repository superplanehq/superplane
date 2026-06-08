import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

type Recipient = { type: string; user?: string; role?: string; group?: string };

export function RecipientsLabel({ recipients }: { recipients?: Recipient[] }) {
  if (!recipients || recipients.length === 0) {
    return null;
  }

  const count = recipients.length;
  const label = `${count} recipient${count > 1 ? "s" : ""}`;

  const counts = { user: 0, role: 0, group: 0 };
  for (const r of recipients) {
    if (r.type in counts) counts[r.type as keyof typeof counts]++;
  }

  const parts: string[] = [];
  if (counts.user > 0) parts.push(`${counts.user} user${counts.user > 1 ? "s" : ""}`);
  if (counts.role > 0) parts.push(`${counts.role} role${counts.role > 1 ? "s" : ""}`);
  if (counts.group > 0) parts.push(`${counts.group} group${counts.group > 1 ? "s" : ""}`);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="cursor-default underline underline-offset-3 decoration-dotted decoration-1">{label}</span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{parts.join(", ")}</TooltipContent>
    </Tooltip>
  );
}
