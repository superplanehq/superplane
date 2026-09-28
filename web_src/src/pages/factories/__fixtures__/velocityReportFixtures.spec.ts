import { describe, expect, it } from "bun:test";

import {
  DEFAULT_FACTORY_VELOCITY,
  EARLY_USAGE_FACTORY_VELOCITY,
  EMPTY_FACTORY_VELOCITY,
  paginateVelocityPeople,
  PEOPLE_LOAD_MORE_FACTORY_VELOCITY,
  PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
} from "./velocityReportFixtures";

const PERIODS = [7, 14, 30] as const;

const SCENARIOS = {
  default: DEFAULT_FACTORY_VELOCITY,
  empty: EMPTY_FACTORY_VELOCITY,
  earlyUsage: EARLY_USAGE_FACTORY_VELOCITY,
  peopleLoadMore: PEOPLE_LOAD_MORE_FACTORY_VELOCITY,
  peopleSyncPending: PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
};

describe("velocity report fixtures", () => {
  it("returns a series of the requested length for every Storybook scenario", () => {
    for (const [name, byPeriod] of Object.entries(SCENARIOS)) {
      for (const periodDays of PERIODS) {
        expect(byPeriod[periodDays]?.points, `${name} ${periodDays}d`).toHaveLength(periodDays);
      }
    }
  });
});

describe("paginateVelocityPeople", () => {
  it("breaks equal task waste by total merges, then SuperPlane merges, then name", () => {
    const report = {
      people: [
        { id: "carol", name: "Carol", factoryWaste: 3, authoredMerged: 1, factoryMerged: 0 },
        { id: "bob", name: "Bob", factoryWaste: 3, authoredMerged: 2, factoryMerged: 1 },
        { id: "ada", name: "Ada", factoryWaste: 1, authoredMerged: 9, factoryMerged: 9 },
        { id: "bea", name: "Bea", factoryWaste: 3, authoredMerged: 1, factoryMerged: 2 },
        { id: "ann", name: "Ann", factoryWaste: 3, authoredMerged: 1, factoryMerged: 2 },
      ],
    };

    const namesFor = (direction: string) => {
      const url = new URL("http://localhost/velocity");
      url.searchParams.set("peopleSort", "PEOPLE_SORT_FACTORY_WASTE");
      url.searchParams.set("peopleSortDirection", direction);
      url.searchParams.set("peoplePageSize", "10");
      return paginateVelocityPeople(report, url).people?.map((person) => person.name);
    };

    expect(namesFor("SORT_DIRECTION_DESC")).toEqual(["Ann", "Bea", "Bob", "Carol", "Ada"]);
    expect(namesFor("SORT_DIRECTION_ASC")).toEqual(["Ada", "Ann", "Bea", "Bob", "Carol"]);
  });
});
