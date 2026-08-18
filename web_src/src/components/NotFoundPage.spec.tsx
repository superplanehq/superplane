import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { NotFoundPage } from "./NotFoundPage";

describe("NotFoundPage", () => {
  it("renders the flight-themed missing page", () => {
    render(
      <NotFoundPage
        description="This plane has left the control plane."
        actionLabel="Return to the hangar"
        showFlightAnimation
      />,
    );

    expect(screen.getByRole("heading", { name: "404" })).toBeInTheDocument();
    expect(screen.getByText("This plane has left the control plane.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Return to the SuperPlane home page" })).toHaveAttribute("href", "/");
    expect(screen.getByTestId("flight-path-illustration")).toHaveAttribute("aria-hidden", "true");
  });
});
