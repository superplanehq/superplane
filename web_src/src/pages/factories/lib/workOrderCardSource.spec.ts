import { describe, expect, it } from "bun:test";

import type { FactoriesWorkOrder } from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import superplaneIcon from "@/assets/superplane.svg";

import { CREATED_MANUALLY } from "../pages/work-order-split-run/splitRunSource";
import { workOrderCardSource, workOrderCardSourceLabel } from "./workOrderCardSource";

const baseOrder: FactoriesWorkOrder = {
  id: "wo-1",
  number: "12",
  title: "Ship refund retries",
  state: "STATE_OPEN",
};

describe("workOrderCardSource", () => {
  it("returns the product logo for a manual task", () => {
    expect(workOrderCardSource(baseOrder)).toEqual({
      name: CREATED_MANUALLY,
      iconSrc: superplaneIcon,
      iconAlt: "SuperPlane",
    });
  });

  it("carries the creator name from the order", () => {
    expect(
      workOrderCardSource({
        ...baseOrder,
        createdBy: { user: { id: "user-1", name: "Ada Lovelace" } },
      }),
    ).toEqual({
      name: CREATED_MANUALLY,
      iconSrc: superplaneIcon,
      iconAlt: "SuperPlane",
      creatorName: "Ada Lovelace",
    });
  });

  it("omits creatorName when the order has no user name", () => {
    expect(
      workOrderCardSource({
        ...baseOrder,
        createdBy: { user: { id: "user-1" } },
      })?.creatorName,
    ).toBeUndefined();
  });

  it("returns an intake source for a GitHub origin", () => {
    expect(
      workOrderCardSource({
        ...baseOrder,
        origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
      }),
    ).toEqual({
      name: "GitHub issues",
      iconSrc: githubIcon,
      iconAlt: "GitHub",
      ticket: { label: "acme/payments#12", href: "https://github.com/acme/payments/issues/12" },
    });
  });
});

describe("workOrderCardSourceLabel", () => {
  it("names a manual source with the creator when present", () => {
    expect(
      workOrderCardSourceLabel({
        name: CREATED_MANUALLY,
        iconSrc: superplaneIcon,
        iconAlt: "SuperPlane",
        creatorName: "Ada Lovelace",
      }),
    ).toBe("Created manually by Ada Lovelace");
  });

  it("falls back to Created manually when the creator is missing", () => {
    expect(
      workOrderCardSourceLabel({
        name: CREATED_MANUALLY,
        iconSrc: superplaneIcon,
        iconAlt: "SuperPlane",
      }),
    ).toBe(CREATED_MANUALLY);
  });

  it("keeps intake labels unchanged", () => {
    expect(
      workOrderCardSourceLabel({
        name: "GitHub issues",
        iconSrc: githubIcon,
        iconAlt: "GitHub",
        ticket: { label: "acme/payments#12", href: "https://github.com/acme/payments/issues/12" },
      }),
    ).toBe("GitHub issues acme/payments#12");
  });
});
