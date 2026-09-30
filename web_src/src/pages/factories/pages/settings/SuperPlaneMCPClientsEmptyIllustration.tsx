import { cn } from "@/lib/utils";

/** Zero-state art: editor on the left, workspace on the right, MCP link between. */
export function SuperPlaneMCPClientsEmptyIllustration({ className }: { className?: string }) {
  const stroke = "currentColor";
  return (
    <svg
      viewBox="0 0 200 72"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={cn("mx-auto h-[72px] w-[200px] max-w-full text-muted-foreground", className)}
      aria-hidden
    >
      <rect x="8" y="12" width="56" height="48" rx="8" stroke={stroke} strokeOpacity="0.3" strokeWidth="1.5" />
      <rect x="18" y="28" width="28" height="2.5" rx="1.25" fill={stroke} fillOpacity="0.2" />
      <rect x="18" y="36" width="36" height="2.5" rx="1.25" fill={stroke} fillOpacity="0.14" />
      <rect x="18" y="44" width="22" height="2.5" rx="1.25" fill={stroke} fillOpacity="0.14" />

      <rect x="136" y="12" width="56" height="48" rx="8" stroke={stroke} strokeOpacity="0.3" strokeWidth="1.5" />
      <rect
        x="146"
        y="24"
        width="36"
        height="24"
        rx="5"
        fill={stroke}
        fillOpacity="0.06"
        stroke={stroke}
        strokeOpacity="0.22"
      />
      <rect x="152" y="30" width="24" height="2" rx="1" fill={stroke} fillOpacity="0.18" />
      <rect x="152" y="36" width="18" height="2" rx="1" fill={stroke} fillOpacity="0.12" />

      <line
        x1="68"
        y1="36"
        x2="88"
        y2="36"
        stroke={stroke}
        strokeOpacity="0.28"
        strokeWidth="1.5"
        strokeDasharray="3 4"
        strokeLinecap="round"
      />
      <circle cx="100" cy="36" r="11" fill="var(--background)" stroke={stroke} strokeOpacity="0.32" strokeWidth="1.5" />
      <path d="M96 36h8M100 32v8" stroke={stroke} strokeOpacity="0.55" strokeWidth="1.5" strokeLinecap="round" />
      <line
        x1="112"
        y1="36"
        x2="132"
        y2="36"
        stroke={stroke}
        strokeOpacity="0.28"
        strokeWidth="1.5"
        strokeDasharray="3 4"
        strokeLinecap="round"
      />
    </svg>
  );
}
