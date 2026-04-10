import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

describe("DialogContent accessibility", () => {
  it("adds a hidden fallback title when a dialog title is missing", () => {
    render(
      <Dialog open>
        <DialogContent>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByText("Dialog")).toBeInTheDocument();
    expect(screen.getByText("Body")).toBeInTheDocument();
  });

  it("does not add the fallback title when a dialog title is already present", () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>Explicit title</DialogTitle>
          <div>Body</div>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByText("Explicit title")).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });
});
