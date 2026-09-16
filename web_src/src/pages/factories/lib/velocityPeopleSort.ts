import type {
  DescribeFactoryVelocityRequestPeopleSort,
  DescribeFactoryVelocityRequestSortDirection,
} from "@/api-client";

/**
 * Columns the People table can sort by. Named to match `VelocityPerson`
 * fields (`"total"` stands in for `authoredMerged + factoryMerged`), and
 * mapped one-to-one onto the backend's `PeopleSort` enum below.
 */
export type PeopleSortKey = "total" | "factoryMerged" | "authoredMerged" | "medianCycleHours" | "costUsd";

export type PeopleSortDirection = "asc" | "desc";

export const PEOPLE_SORT_DEFAULT_KEY: PeopleSortKey = "total";
export const PEOPLE_SORT_DEFAULT_DIRECTION: PeopleSortDirection = "desc";

/** Rows fetched for the first People page. */
export const PEOPLE_FIRST_PAGE_SIZE = 5;

/** Rows fetched on every "Show more" click after the first page. */
export const PEOPLE_LOAD_MORE_SIZE = 20;

/** Page size for a given offset: 6 on the first page, 20 after that. */
export function peoplePageSizeForOffset(offset: number): number {
  return offset <= 0 ? PEOPLE_FIRST_PAGE_SIZE : PEOPLE_LOAD_MORE_SIZE;
}

/** Offset of the next People page after the page that starts at `offset`. */
export function nextPeopleOffset(offset: number): number {
  return offset + peoplePageSizeForOffset(offset);
}

const PEOPLE_SORT_PARAM: Record<PeopleSortKey, DescribeFactoryVelocityRequestPeopleSort> = {
  total: "PEOPLE_SORT_TOTAL",
  factoryMerged: "PEOPLE_SORT_FACTORY_MERGED",
  authoredMerged: "PEOPLE_SORT_AUTHORED_MERGED",
  medianCycleHours: "PEOPLE_SORT_MEDIAN_CYCLE_HOURS",
  costUsd: "PEOPLE_SORT_COST_USD",
};

const PEOPLE_SORT_DIRECTION_PARAM: Record<PeopleSortDirection, DescribeFactoryVelocityRequestSortDirection> = {
  asc: "SORT_DIRECTION_ASC",
  desc: "SORT_DIRECTION_DESC",
};

/** Translates a People sort key into the request's wire enum value. */
export function peopleSortParam(key: PeopleSortKey): DescribeFactoryVelocityRequestPeopleSort {
  return PEOPLE_SORT_PARAM[key];
}

/** Translates a People sort direction into the request's wire enum value. */
export function peopleSortDirectionParam(direction: PeopleSortDirection): DescribeFactoryVelocityRequestSortDirection {
  return PEOPLE_SORT_DIRECTION_PARAM[direction];
}
