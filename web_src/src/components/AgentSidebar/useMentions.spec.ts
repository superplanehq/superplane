import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useMentions } from "./useMentions";

describe("useMentions", () => {
  describe("detectTrigger (via hook)", () => {
    it("no trigger when no @ present", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("hello world");
        result.current.setCursorPos(11);
      });
      expect(result.current.showDropdown).toBe(false);
    });

    it("trigger active after @ at start of text", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      expect(result.current.showDropdown).toBe(true);
      expect(result.current.filter).toBe("");
    });

    it("trigger active after @ preceded by space", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("hello @");
        result.current.setCursorPos(7);
      });
      expect(result.current.showDropdown).toBe(true);
    });

    it("no trigger for @ in middle of word", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("email@example");
        result.current.setCursorPos(13);
      });
      expect(result.current.showDropdown).toBe(false);
    });

    it("filter includes text after @", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@PR");
        result.current.setCursorPos(3);
      });
      expect(result.current.showDropdown).toBe(true);
      expect(result.current.filter).toBe("PR");
    });

    it("spaces allowed in filter", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@PR Op");
        result.current.setCursorPos(6);
      });
      expect(result.current.showDropdown).toBe(true);
      expect(result.current.filter).toBe("PR Op");
    });

    it("newline terminates trigger", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@hello\nworld");
        result.current.setCursorPos(12);
      });
      expect(result.current.showDropdown).toBe(false);
    });
  });

  describe("insertMention", () => {
    it("inserts @Label at trigger position", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "MyNode" });
      });
      expect(result.current.value).toBe("@MyNode ");
    });

    it("returns correct new cursor position", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("hello @");
        result.current.setCursorPos(7);
      });
      let newPos: number | undefined;
      act(() => {
        newPos = result.current.insertMention({ type: "node", id: "n1", label: "Test" });
      });
      expect(newPos).toBe(12); // "hello @Test ".length = 12
    });

    it("tracks mention with startIndex", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "A" });
      });
      expect(result.current.mentions).toHaveLength(1);
      expect(result.current.mentions[0].startIndex).toBe(0);
      expect(result.current.mentions[0].id).toBe("n1");
    });

    it("inserting before existing mention shifts its startIndex", () => {
      const { result } = renderHook(() => useMentions());
      // Insert first mention
      act(() => {
        result.current.setValue("hello @");
        result.current.setCursorPos(7);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "First" });
      });
      // value is "hello @First ", mention at index 6
      const firstStartBefore = result.current.mentions.find((m) => m.id === "n1")!.startIndex;
      // Insert at start: modify value to have @ at position 0
      act(() => {
        result.current.setValue("@" + result.current.value);
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n2", label: "Second" });
      });
      // The second mention is at position 0, first should have shifted right
      const secondMention = result.current.mentions.find((m) => m.id === "n2");
      expect(secondMention!.startIndex).toBe(0);
      // First mention still tracked (its text is still there shifted)
      const firstMention = result.current.mentions.find((m) => m.id === "n1");
      if (firstMention) {
        expect(firstMention.startIndex).toBeGreaterThan(firstStartBefore);
      }
    });
  });

  describe("getMarkdown", () => {
    it("replaces tracked mentions with [Label](node:id) format", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "MyNode" });
      });
      expect(result.current.getMarkdown()).toBe("[MyNode](node:n1) ");
    });

    it("replaces run mentions with [Label](run:id) format", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "run", id: "r1", label: "Run #abc123" });
      });
      expect(result.current.getMarkdown()).toBe("[Run #abc123](run:r1) ");
    });

    it("leaves untracked @text as-is", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("hey @someone");
        result.current.setCursorPos(12);
      });
      expect(result.current.getMarkdown()).toBe("hey @someone");
    });

    it("handles multiple mentions correctly", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "A" });
      });
      // value is now "@A "
      act(() => {
        result.current.setValue(result.current.value + "@");
        result.current.setCursorPos(result.current.value.length + 1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n2", label: "B" });
      });
      const md = result.current.getMarkdown();
      expect(md).toContain("[A](node:n1)");
      expect(md).toContain("[B](node:n2)");
    });
  });

  describe("pruneMentions (via setValue)", () => {
    it("removing a mention's text prunes it from tracking", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "Test" });
      });
      expect(result.current.mentions).toHaveLength(1);
      act(() => {
        result.current.setValue("something else");
      });
      expect(result.current.mentions).toHaveLength(0);
    });

    it("editing text that breaks a mention removes it", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "Hello" });
      });
      // Change @Hello to @Hell (partial)
      act(() => {
        result.current.setValue("@Hell ");
      });
      expect(result.current.mentions).toHaveLength(0);
    });
  });

  describe("snapshot/restore", () => {
    it("snapshot saves current state and restore recovers it", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.insertMention({ type: "node", id: "n1", label: "X" });
      });
      act(() => {
        result.current.snapshot();
      });
      // Manually reset without using clear (which nulls snapshot)
      act(() => {
        result.current.setValue("");
        result.current.setCursorPos(0);
      });
      expect(result.current.value).toBe("");
      act(() => {
        result.current.restore();
      });
      expect(result.current.value).toBe("@X ");
      expect(result.current.mentions).toHaveLength(1);
    });

    it("restore sets cursor to end of text", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("hello");
        result.current.setCursorPos(5);
      });
      act(() => {
        result.current.snapshot();
      });
      act(() => {
        result.current.setValue("");
        result.current.setCursorPos(0);
      });
      act(() => {
        result.current.restore();
      });
      expect(result.current.cursorPos).toBe(5);
    });
  });

  describe("dismiss", () => {
    it("dismiss() hides dropdown", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      expect(result.current.showDropdown).toBe(true);
      act(() => {
        result.current.dismiss();
      });
      expect(result.current.showDropdown).toBe(false);
    });

    it("typing after dismiss shows dropdown again", () => {
      const { result } = renderHook(() => useMentions());
      act(() => {
        result.current.setValue("@");
        result.current.setCursorPos(1);
      });
      act(() => {
        result.current.dismiss();
      });
      expect(result.current.showDropdown).toBe(false);
      act(() => {
        result.current.setValue("@N");
        result.current.setCursorPos(2);
      });
      expect(result.current.showDropdown).toBe(true);
    });
  });
});
