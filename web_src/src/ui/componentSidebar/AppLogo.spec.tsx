import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppLogo } from "./AppLogo";
import { getIconThemeTreatment, getIntegrationIconSrc } from "./integrationIconMaps";

describe("getIconThemeTreatment", () => {
  it("marks monochrome logos for inversion on dark surfaces", () => {
    for (const name of ["github", "semaphore", "circleci"]) {
      const src = getIntegrationIconSrc(name)!;
      expect(getIconThemeTreatment(src)).toEqual({ invertInDark: true });
    }
  });

  it("gives brand-colored logos a dedicated dark asset instead of inverting", () => {
    const treatment = getIconThemeTreatment(getIntegrationIconSrc("aws"));
    expect(treatment.invertInDark).toBeUndefined();
    expect(treatment.darkSrc).toBeTruthy();
    expect(treatment.darkSrc).not.toEqual(getIntegrationIconSrc("aws"));
  });

  it("leaves theme-agnostic logos untouched", () => {
    expect(getIconThemeTreatment(getIntegrationIconSrc("slack"))).toEqual({});
    expect(getIconThemeTreatment(undefined)).toEqual({});
  });
});

describe("AppLogo", () => {
  it("applies the dark-invert filter to monochrome logos", () => {
    const src = getIntegrationIconSrc("github")!;
    const { container } = render(<AppLogo src={src} className="h-4 w-4" />);
    const imgs = container.querySelectorAll("img");
    expect(imgs).toHaveLength(1);
    expect(imgs[0].className).toContain("dark:invert");
  });

  it("renders a CSS-swapped pair for brand-colored logos with a dark asset", () => {
    const src = getIntegrationIconSrc("aws")!;
    const { container } = render(<AppLogo src={src} className="h-4 w-4" />);
    const imgs = container.querySelectorAll("img");
    expect(imgs).toHaveLength(2);
    expect(imgs[0].className).toContain("dark:hidden");
    expect(imgs[1].className).toContain("hidden");
    expect(imgs[1].className).toContain("dark:block");
    expect(imgs[1].getAttribute("src")).not.toEqual(imgs[0].getAttribute("src"));
  });

  it("renders a single unmodified image for theme-agnostic logos", () => {
    const src = getIntegrationIconSrc("slack")!;
    const { container } = render(<AppLogo src={src} className="h-4 w-4" />);
    const imgs = container.querySelectorAll("img");
    expect(imgs).toHaveLength(1);
    expect(imgs[0].className).not.toContain("invert");
  });
});
