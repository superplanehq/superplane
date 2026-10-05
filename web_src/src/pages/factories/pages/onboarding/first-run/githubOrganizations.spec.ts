import { describe, expect, it } from "bun:test";

import {
  findOrganization,
  ownerOfRepository,
  repositoriesInOrganization,
  repositoryOrganizations,
} from "./githubOrganizations";

const repositories = ["superplanehq/superplane", "forestileao/test", "Forestileao/instabot"];

describe("repositoryOrganizations", () => {
  it("lists each owner once, in name order", () => {
    expect(repositoryOrganizations(repositories)).toEqual(["forestileao", "superplanehq"]);
  });

  it("returns no organizations for no repositories", () => {
    expect(repositoryOrganizations([])).toEqual([]);
  });
});

describe("repositoriesInOrganization", () => {
  it("keeps only the repositories of the organization, without case differences", () => {
    expect(repositoriesInOrganization(repositories, "forestileao")).toEqual([
      "forestileao/test",
      "Forestileao/instabot",
    ]);
  });
});

describe("findOrganization", () => {
  it("finds the listed organization without case differences", () => {
    expect(findOrganization(["forestileao", "superplanehq"], "SuperPlaneHQ")).toBe("superplanehq");
  });

  it("returns null when the list does not have the organization", () => {
    expect(findOrganization(["forestileao"], "superplanehq")).toBeNull();
    expect(findOrganization(["forestileao"], null)).toBeNull();
  });
});

describe("ownerOfRepository", () => {
  it("returns the owner part of the repository name", () => {
    expect(ownerOfRepository("superplanehq/superplane")).toBe("superplanehq");
    expect(ownerOfRepository(null)).toBeNull();
  });
});
