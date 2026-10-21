import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import { lineIntakeSourceById, type LineIntakeSourceId } from "./lineIntakeModel";

export function IntakeSourceTitleLogo({ sourceId }: { sourceId: LineIntakeSourceId }) {
  const source = lineIntakeSourceById(sourceId);
  if (!source?.iconSrc) {
    return null;
  }

  return (
    <img
      src={source.iconSrc}
      alt=""
      data-testid="intake-source-settings-title-icon"
      className={cn(
        "size-5 shrink-0 object-contain",
        source.iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
        logoDarkInvertClass(source.iconSrc),
      )}
    />
  );
}
