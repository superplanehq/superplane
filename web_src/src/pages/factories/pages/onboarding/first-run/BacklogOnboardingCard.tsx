import { FIRST_RUN_COPY } from "./firstRunCopy";

/** First-run backlog zero state. Centered copy, not a ticket card. */
export function BacklogOnboardingCard() {
  const copy = FIRST_RUN_COPY.board;

  return (
    <article
      className="flex min-h-40 flex-1 flex-col items-center justify-center px-3 py-10 text-center"
      data-testid="backlog-onboarding-card"
    >
      <h3 className="max-w-[14rem] text-[15px] font-semibold leading-snug tracking-[-0.03em] text-balance text-foreground">
        {copy.backlogHintTitle}
      </h3>
      <p className="workspace-body-text mt-2 max-w-[15rem] text-pretty text-muted-foreground">{copy.backlogHintBody}</p>
    </article>
  );
}
