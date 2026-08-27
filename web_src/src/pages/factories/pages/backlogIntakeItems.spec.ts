import { describe, expect, it } from "vitest";

import type { FactoriesFactoryIntake } from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import {
  listBacklogIntakeSources,
  searchBacklogIntakeItems,
  searchPlaceholderForIntake,
  type BacklogIntakeItem,
} from "./backlogIntakeItems";

const github: FactoriesFactoryIntake = {
  id: "intake-github",
  name: "GitHub issues",
  source: "SOURCE_GITHUB_ISSUES",
};
const sentry: FactoriesFactoryIntake = {
  id: "intake-sentry",
  name: "Sentry exceptions",
  source: "SOURCE_SENTRY_EXCEPTIONS",
};
const linear: FactoriesFactoryIntake = {
  id: "intake-linear",
  name: "Linear issues",
  source: "SOURCE_UNSPECIFIED",
};

const items: BacklogIntakeItem[] = [
  {
    id: "gh-1",
    intakeId: "intake-github",
    key: "#12",
    title: "Handle duplicate refunds",
    body: "Retrying a refund posts twice.",
  },
  {
    id: "gh-2",
    intakeId: "intake-github",
    key: "#13",
    title: "Upgrade Node 20",
    body: "Base image is stale.",
  },
  {
    id: "se-1",
    intakeId: "intake-sentry",
    key: "PROJ-9",
    title: "Null pointer in checkout",
    body: "TypeError on missing cart.",
  },
  {
    id: "ln-1",
    intakeId: "intake-linear",
    key: "LIN-4",
    title: "Triage customer bugs",
    body: "Move bugs from the inbox.",
  },
];

describe("searchBacklogIntakeItems", () => {
  it("groups latest items by configured intake", () => {
    const groups = searchBacklogIntakeItems({
      intakes: [github, sentry, linear],
      catalog: { items, iconSrcByIntakeId: { "intake-linear": "/linear.svg" } },
      query: "",
    });

    expect(groups.map((group) => group.name)).toEqual(["GitHub issues", "Sentry exceptions", "Linear issues"]);
    expect(groups[0]?.items.map((item) => item.key)).toEqual(["#12", "#13"]);
    expect(groups[2]?.iconSrc).toBe("/linear.svg");
  });

  it("returns every matching item so the menu can page them on scroll", () => {
    const many: BacklogIntakeItem[] = Array.from({ length: 8 }, (_, index) => ({
      id: `gh-${index}`,
      intakeId: "intake-github",
      key: `#${index}`,
      title: `Issue ${index}`,
      body: "",
    }));

    const groups = searchBacklogIntakeItems({
      intakes: [github],
      catalog: { items: many },
      query: "",
    });

    expect(groups[0]?.items).toHaveLength(8);
    expect(groups[0]?.items[0]?.key).toBe("#0");
  });

  it("searches across intake groups by title, key, and body", () => {
    const groups = searchBacklogIntakeItems({
      intakes: [github, sentry, linear],
      catalog: { items },
      query: "null pointer",
    });

    expect(groups).toHaveLength(1);
    expect(groups[0]?.name).toBe("Sentry exceptions");
    expect(groups[0]?.items[0]?.key).toBe("PROJ-9");
  });

  it("omits intakes that have no matching items", () => {
    const groups = searchBacklogIntakeItems({
      intakes: [github, sentry],
      catalog: { items },
      query: "LIN-4",
    });

    expect(groups).toEqual([]);
  });

  it("lists every configured intake as a search source", () => {
    const sources = listBacklogIntakeSources({
      intakes: [github, sentry, linear],
      catalog: { iconSrcByIntakeId: { "intake-linear": "/linear.svg" } },
    });

    expect(sources.map((source) => source.name)).toEqual(["GitHub issues", "Sentry exceptions", "Linear issues"]);
    expect(sources[0]?.iconSrc).toBe(githubIcon);
    expect(sources[0]?.iconAlt).toBe("GitHub");
    expect(sources[1]?.iconSrc).toBe(sentryIcon);
    expect(sources[2]?.iconSrc).toBe("/linear.svg");
  });

  it("uses the GitHub icon when the intake has no catalog override", () => {
    const sources = listBacklogIntakeSources({ intakes: [github] });

    expect(sources).toEqual([
      {
        intakeId: "intake-github",
        name: "GitHub issues",
        iconSrc: githubIcon,
        iconAlt: "GitHub",
      },
    ]);
  });

  it("builds a source-specific search placeholder", () => {
    expect(searchPlaceholderForIntake("GitHub issues")).toBe("Import from GitHub issue");
    expect(searchPlaceholderForIntake("Sentry exceptions")).toBe("Import from Sentry exception");
    expect(searchPlaceholderForIntake("PagerDuty incidents")).toBe("Import from PagerDuty incident");
  });
});
