import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { BacklogOnboardingCard } from "./BacklogOnboardingCard";
import { FIRST_RUN_COPY } from "./firstRunCopy";

describe("BacklogOnboardingCard", () => {
  it("explains that analyzed intake tickets appear in the backlog for review", () => {
    render(<BacklogOnboardingCard />);

    const card = screen.getByTestId("backlog-onboarding-card");
    expect(card).toHaveTextContent(FIRST_RUN_COPY.board.backlogHintTitle);
    expect(card).toHaveTextContent(FIRST_RUN_COPY.board.backlogHintBody);
    expect(card).not.toHaveTextContent(/candidates from intake/i);
    expect(card).not.toHaveTextContent(/work order/i);
  });
});
