import { describe, expect, it } from "vitest";

import {
  PEOPLE_FIRST_PAGE_SIZE,
  PEOPLE_LOAD_MORE_SIZE,
  nextPeopleOffset,
  peoplePageSizeForOffset,
} from "./velocityPeopleSort";

describe("peoplePageSizeForOffset", () => {
  it("asks for 5 rows on the first page and 20 on every Show more page", () => {
    expect(peoplePageSizeForOffset(0)).toBe(PEOPLE_FIRST_PAGE_SIZE);
    expect(peoplePageSizeForOffset(PEOPLE_FIRST_PAGE_SIZE)).toBe(PEOPLE_LOAD_MORE_SIZE);
    expect(peoplePageSizeForOffset(PEOPLE_FIRST_PAGE_SIZE + PEOPLE_LOAD_MORE_SIZE)).toBe(PEOPLE_LOAD_MORE_SIZE);
  });
});

describe("nextPeopleOffset", () => {
  it("walks 5, then 25, then 45", () => {
    expect(nextPeopleOffset(0)).toBe(5);
    expect(nextPeopleOffset(5)).toBe(25);
    expect(nextPeopleOffset(25)).toBe(45);
  });
});
