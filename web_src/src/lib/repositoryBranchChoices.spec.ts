import { describe, expect, it } from "bun:test";

import { repositoryBranchChoices } from "./repositoryBranchChoices";

describe("repositoryBranchChoices", () => {
  it("lists common default branches first and keeps an unlisted saved branch", () => {
    expect(
      repositoryBranchChoices(["dev", "zebra", "master", "bugfix", "Develop", "Main", "trunk"], "feature/login"),
    ).toEqual(["Main", "master", "Develop", "dev", "trunk", "bugfix", "feature/login", "zebra"]);
  });
});
