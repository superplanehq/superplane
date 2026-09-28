import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { defaultRiskScoreCategories, type RiskScoreCategory } from "./riskScoreCategories";
import { RiskScoreCategoryEditor } from "./RiskScoreCategoryEditor";

function Editor() {
  const [categories, setCategories] = useState<RiskScoreCategory[]>(defaultRiskScoreCategories());
  return <RiskScoreCategoryEditor categories={categories} onChange={setCategories} />;
}

describe("RiskScoreCategoryEditor", () => {
  it("colors each score", () => {
    render(<Editor />);

    expect(screen.getByTestId("risk-score-category-database-additive")).toHaveTextContent("Moderate Risk");
    expect(screen.getByTestId("risk-score-category-database-additive")).toHaveStyle({ color: "#b45309" });
    expect(screen.getByTestId("risk-score-category-authorization")).toHaveTextContent("High Risk");
    expect(screen.getByTestId("risk-score-category-authorization")).toHaveStyle({ color: "#b91c1c" });
    expect(screen.getByTestId("risk-score-category-documentation")).toHaveStyle({ color: "#047857" });
  });

  it("adds and removes a category", async () => {
    const user = userEvent.setup();
    render(<Editor />);

    await user.click(screen.getByTestId("risk-score-category-remove-documentation"));
    expect(screen.queryByText("Documentation only")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("risk-score-category-add-more"));
    await user.type(screen.getByTestId("risk-score-category-name"), "Cache changes");
    await user.click(screen.getByTestId("risk-score-category-add"));

    expect(screen.getByText("Cache changes")).toBeInTheDocument();
    expect(screen.getByTestId("risk-score-category-custom-1")).toHaveTextContent("Moderate Risk");
  });

  it("rejects a name that contains an equals sign", async () => {
    const user = userEvent.setup();
    render(<Editor />);

    await user.click(screen.getByTestId("risk-score-category-add-more"));
    await user.type(screen.getByTestId("risk-score-category-name"), "Connection = pool");

    expect(screen.getByTestId("risk-score-category-add")).toBeDisabled();
    expect(screen.queryByText("Connection = pool")).not.toBeInTheDocument();
  });
});
