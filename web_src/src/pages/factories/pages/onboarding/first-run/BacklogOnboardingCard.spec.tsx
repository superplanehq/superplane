import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { BacklogOnboardingCard } from "./BacklogOnboardingCard";
import { FIRST_RUN_COPY } from "./firstRunCopy";

describe("BacklogOnboardingCard", () => {
  it("explains that new issues land in the backlog as work orders", () => {
    render(<BacklogOnboardingCard />);

    const card = screen.getByTestId("backlog-onboarding-card");
    expect(card).toHaveTextContent(FIRST_RUN_COPY.board.backlogHintTitle);
    expect(card).toHaveTextContent(FIRST_RUN_COPY.board.backlogHintBody);
    expect(card).not.toHaveTextContent(/candidates from intake/i);
  });
});
