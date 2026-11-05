import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryIntake } from "@/api-client";

import { EMPTY_LINEAR_INTAKE_SCOPE } from "../linearIntakeScope";
import { provisionOnboardingIntake } from "./onboardingProvision";

describe("provisionOnboardingIntake for Linear", () => {
  it("creates a Linear intake for a team with no projects", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    const intake = await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      updateIntake: vi.fn(),
      deleteIntake,
      issuesChoice: "linear",
      linear: { integrationId: "linear-1", scope: { ...EMPTY_LINEAR_INTAKE_SCOPE, teamId: "team-1" } },
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      settings: {
        linearTeamId: "team-1",
        linearProjectIds: [],
        linearIncludeIssuesWithoutProject: true,
        linearLabels: [],
      },
    });
    expect(intake?.id).toBe("intake-linear");
  });

  it("sends the project filter and the no-project choice with the team", async () => {
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear" } as FactoriesFactoryIntake);

    await provisionOnboardingIntake({
      listIntakes: vi.fn().mockResolvedValue([]),
      createIntake,
      updateIntake: vi.fn(),
      deleteIntake: vi.fn(),
      issuesChoice: "linear",
      linear: {
        integrationId: "linear-1",
        scope: { teamId: "team-1", projectIds: ["project-1"], includeIssuesWithoutProject: false, labels: ["bug"] },
      },
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      settings: {
        linearTeamId: "team-1",
        linearProjectIds: ["project-1"],
        linearIncludeIssuesWithoutProject: false,
        linearLabels: ["bug"],
      },
    });
  });

  it("keeps the Linear intake that a retried finish already created for the same scope", async () => {
    const existing = {
      id: "intake-linear",
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      settings: {
        linearTeamId: "team-1",
        linearProjectIds: ["project-2", "project-1"],
        linearIncludeIssuesWithoutProject: false,
        linearLabels: [],
      },
    } as FactoriesFactoryIntake;
    const createIntake = vi.fn();
    const deleteIntake = vi.fn();

    const intake = await provisionOnboardingIntake({
      listIntakes: vi.fn().mockResolvedValue([existing]),
      createIntake,
      updateIntake: vi.fn(),
      deleteIntake,
      issuesChoice: "linear",
      linear: {
        integrationId: "linear-1",
        scope: {
          teamId: "team-1",
          projectIds: ["project-1", "project-2"],
          includeIssuesWithoutProject: false,
          labels: [],
        },
      },
    });

    expect(intake).toBe(existing);
    expect(createIntake).not.toHaveBeenCalled();
    expect(deleteIntake).not.toHaveBeenCalled();
  });

  it("replaces a Linear intake when the retry chooses another team", async () => {
    const existing = {
      id: "intake-linear",
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      settings: { linearTeamId: "team-1" },
    } as FactoriesFactoryIntake;
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear-2" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn().mockResolvedValue({});

    await provisionOnboardingIntake({
      listIntakes: vi.fn().mockResolvedValue([existing]),
      createIntake,
      updateIntake: vi.fn(),
      deleteIntake,
      issuesChoice: "linear",
      linear: { integrationId: "linear-1", scope: { ...EMPTY_LINEAR_INTAKE_SCOPE, teamId: "team-2" } },
    });

    expect(deleteIntake).toHaveBeenCalledWith("intake-linear");
    expect(createIntake).toHaveBeenCalledTimes(1);
  });

  it("applies the automatic intake choice to a reused Linear intake", async () => {
    const existing = {
      id: "intake-linear",
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      settings: { linearTeamId: "team-1" },
    } as FactoriesFactoryIntake;
    const updateIntake = vi.fn().mockResolvedValue({ ...existing, paused: true });

    await provisionOnboardingIntake({
      listIntakes: vi.fn().mockResolvedValue([existing]),
      createIntake: vi.fn(),
      updateIntake,
      deleteIntake: vi.fn(),
      paused: true,
      issuesChoice: "linear",
      linear: { integrationId: "linear-1", scope: { ...EMPTY_LINEAR_INTAKE_SCOPE, teamId: "team-1" } },
    });

    expect(updateIntake).toHaveBeenCalledWith({ intakeId: "intake-linear", paused: true });
  });

  it("creates a paused Linear intake when automatic intake is off", async () => {
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear" } as FactoriesFactoryIntake);

    await provisionOnboardingIntake({
      listIntakes: vi.fn().mockResolvedValue([]),
      createIntake,
      updateIntake: vi.fn(),
      deleteIntake: vi.fn(),
      paused: true,
      issuesChoice: "linear",
      linear: { integrationId: "linear-1", scope: { ...EMPTY_LINEAR_INTAKE_SCOPE, teamId: "team-1" } },
    });

    expect(createIntake).toHaveBeenCalledWith(expect.objectContaining({ paused: true }));
  });
});
