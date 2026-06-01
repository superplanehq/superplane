import { generateAppDotGrid } from "@/lib/appDotGrid";
import { cn } from "@/lib/utils";
import { useMemo } from "react";

interface AppDotGridProps {
  seed: string;
}

export function AppDotGrid({ seed }: AppDotGridProps) {
  const dots = useMemo(() => generateAppDotGrid(seed), [seed]);

  return (
    <div className="grid size-7 shrink-0 grid-cols-6 gap-0.5" aria-hidden>
      {dots.map((visible, index) => (
        <span
          key={index}
          className={cn("size-[3px] rounded-full", visible ? "bg-gray-800 dark:bg-gray-300" : "bg-transparent")}
        />
      ))}
    </div>
  );
}
