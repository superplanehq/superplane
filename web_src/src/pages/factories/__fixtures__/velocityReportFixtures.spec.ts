import { describe, expect, it } from "bun:test";

import {
  DEFAULT_FACTORY_VELOCITY,
  EARLY_USAGE_FACTORY_VELOCITY,
  EMPTY_FACTORY_VELOCITY,
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
