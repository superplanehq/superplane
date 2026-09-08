import { describe, expect, it } from "vitest";

import {
  pendingGitHubAccountPicker,
  pendingGitHubInstallPicker,
  pendingGitHubRequestedPicker,
} from "./startDirectGitHubConnect";

describe("pendingGitHubInstallPicker", () => {
  it("returns a pending GitHub connection with a single install", () => {
    expect(
      pendingGitHubAccountPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                startedByGitHubLogin: "forestileao",
                state: "csrf",
                githubApp: { slug: "superplane" },
                pendingInstallations: [{ id: "11", accountLogin: "acme" }],
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({
      id: "int-1",
      state: "csrf",
      appSlug: "superplane",
      authorizeUrl: "",
      githubLogin: "forestileao",
      requestedAccount: "",
      installations: [{ id: "11", accountLogin: "acme" }],
    });
  });

  it("returns a pending GitHub connection with two or more installs", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({ id: "int-1" });
    expect(
      pendingGitHubAccountPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                state: "csrf",
                githubApp: { slug: "superplane" },
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({
      id: "int-1",
      state: "csrf",
      appSlug: "superplane",
      authorizeUrl: "",
      githubLogin: "",
      requestedAccount: "",
      installations: [
        { id: "11", accountLogin: "acme" },
        { id: "22", accountLogin: "octo" },
      ],
    });
  });

  it("returns undefined when there is no picker", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: { state: "pending", browserAction: { method: "GET", url: "https://github.com" } },
          },
        ],
        "user-1",
      ),
    ).toBeUndefined();
  });

  it("returns undefined when the current user is not loaded yet", () => {
    expect(
      pendingGitHubAccountPicker([
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [
                { id: "11", accountLogin: "acme" },
                { id: "22", accountLogin: "octo" },
              ],
            },
          },
        },
      ]),
    ).toBeUndefined();
  });

  it("returns a request-only picker when GitHub returned a request and no ready account", () => {
    expect(
      pendingGitHubRequestedPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                startedByGitHubLogin: "ada",
                installRequested: true,
                installRequestedAccount: "acme",
                state: "csrf",
                githubApp: { slug: "superplane" },
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({
      id: "int-1",
      state: "csrf",
      appSlug: "superplane",
      authorizeUrl: "",
      githubLogin: "ada",
      requestedAccount: "acme",
      installations: [],
    });
    expect(
      pendingGitHubAccountPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                installRequested: true,
                state: "csrf",
              },
            },
          },
        ],
        "user-1",
      ),
    ).toBeUndefined();
  });

  it("returns undefined when the picker belongs to a teammate", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });
});
