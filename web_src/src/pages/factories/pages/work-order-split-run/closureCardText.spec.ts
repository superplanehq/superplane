import { describe, expect, it } from "bun:test";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import { closureCardDescription } from "./closureCardText";

const MERGED: FactoriesFactoryPullRequest = {
  number: "8044",
  title: "feat: Show watched checks on the task card",
  url: "https://github.com/superplanehq/superplane/pull/8044",
  state: "STATE_MERGED",
};

const ALEX = { id: "user-1", name: "Alex Rivera", initials: "AR" };

describe("closureCardDescription", () => {
  it("names the person who merged the pull request and links both", () => {
    expect(closureCardDescription("completed", { actor: ALEX, actorHref: "https://github.com/alex" }, [MERGED])).toBe(
      "Resolved because [Alex Rivera](https://github.com/alex) merged [#8044 feat: Show watched checks on the task card](https://github.com/superplanehq/superplane/pull/8044).",
    );
  });

  it("links the merged pull request when the closer is unknown", () => {
    expect(closureCardDescription("completed", {}, [MERGED])).toBe(
      "Resolved because [#8044 feat: Show watched checks on the task card](https://github.com/superplanehq/superplane/pull/8044) is merged.",
    );
  });

  it("does not treat a closer app name as the person who merged", () => {
    expect(closureCardDescription("completed", { automationName: "PR Closure" }, [MERGED])).toBe(
      "Resolved because [#8044 feat: Show watched checks on the task card](https://github.com/superplanehq/superplane/pull/8044) is merged.",
    );
  });

  it("links a GitHub merger stored on the close event", () => {
    expect(
      closureCardDescription(
        "completed",
        { automationName: "Alex Rivera", automationHref: "https://github.com/alex" },
        [MERGED],
      ),
    ).toBe(
      "Resolved because [Alex Rivera](https://github.com/alex) merged [#8044 feat: Show watched checks on the task card](https://github.com/superplanehq/superplane/pull/8044).",
    );
  });

  it("keeps the close sentence when no pull request is merged", () => {
    expect(closureCardDescription("completed", { actor: ALEX })).toBe("Alex Rivera marked this task as successful.");
    expect(closureCardDescription("completed")).toBe("This task succeeded.");
  });
});
