import { Award, Lock } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { Timestamp } from "@/components/Timestamp";
import type { Achievement } from "@/pages/factories/verification/types";

interface AchievementsGridProps {
  achievements: Achievement[];
}

/**
 * Earned and not-yet-earned achievements. Not-yet-earned entries show what
 * remains to earn them.
 */
export function AchievementsGrid({ achievements }: AchievementsGridProps) {
  return (
    <section className="flex flex-col gap-3" aria-label="Achievements">
      <div className="flex flex-col gap-0.5">
        <h3 className="workspace-section-title text-foreground">Achievements</h3>
        <p className="text-[12px] text-muted-foreground">
          Milestones this factory earned. A later regression does not remove an earned achievement.
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {achievements.map((achievement) => (
          <AchievementTile key={achievement.id} achievement={achievement} />
        ))}
      </div>
    </section>
  );
}

function AchievementTile({ achievement }: { achievement: Achievement }) {
  const earned = achievement.earnedAt != null;
  return (
    <article
      className={cn(factoryCardClassName, "flex flex-col gap-2 p-4", !earned && "opacity-80")}
      aria-label={achievement.name}
    >
      <div className="flex items-start gap-2.5">
        <span
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-full border",
            earned
              ? "border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-400"
              : "border-border bg-background text-muted-foreground",
          )}
        >
          {earned ? <Award className="size-4.5" aria-hidden /> : <Lock className="size-4" aria-hidden />}
        </span>
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13px] font-medium text-foreground">{achievement.name}</span>
          <p className="text-[12px] text-muted-foreground">{achievement.description}</p>
        </div>
      </div>
      <p className="border-t border-border pt-2 text-[12px] text-muted-foreground">
        {earned && achievement.earnedAt ? (
          <>
            Earned <Timestamp date={achievement.earnedAt} display="relative" />
          </>
        ) : (
          (achievement.progressNote ?? "Not earned yet.")
        )}
      </p>
    </article>
  );
}
