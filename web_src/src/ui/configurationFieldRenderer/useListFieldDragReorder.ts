import React from "react";

import { reorderListItems } from "@/lib/reorderListItems";

type DragState = {
  items: unknown[];
  activeIndex: number;
  openAtStart: boolean;
};

export function useListFieldDragReorder({
  items,
  allowReorder,
  useAccordion,
  onChange,
  setOpenItem,
  rowRefs,
}: {
  items: unknown[];
  allowReorder: boolean;
  useAccordion: boolean;
  onChange: (value: unknown) => void;
  setOpenItem: React.Dispatch<React.SetStateAction<string>>;
  rowRefs: React.MutableRefObject<Array<HTMLDivElement | null>>;
}) {
  const itemsRef = React.useRef(items);
  itemsRef.current = items;

  const [dragState, setDragState] = React.useState<DragState | null>(null);
  const dragStateRef = React.useRef<DragState | null>(null);
  dragStateRef.current = dragState;

  const isDragging = dragState !== null;
  const renderedItems = dragState ? dragState.items : items;

  React.useEffect(() => {
    if (!isDragging) {
      return;
    }

    const handleMove = (event: MouseEvent) => {
      setDragState((current) => {
        if (!current) return current;
        for (let i = 0; i < rowRefs.current.length; i++) {
          const el = rowRefs.current[i];
          if (!el) continue;
          const rect = el.getBoundingClientRect();
          if (event.clientY >= rect.top && event.clientY <= rect.bottom) {
            if (i === current.activeIndex) {
              return current;
            }
            const nextItems = reorderListItems(current.items, current.activeIndex, i);
            return { ...current, items: nextItems, activeIndex: i };
          }
        }
        return current;
      });
    };

    const handleUp = () => {
      const current = dragStateRef.current;
      if (!current) return;
      itemsRef.current = current.items;
      onChange(current.items);
      if (useAccordion && current.openAtStart) {
        setOpenItem(String(current.activeIndex));
      }
      setDragState(null);
    };

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = "grabbing";
    document.body.style.userSelect = "none";

    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
    };
  }, [isDragging, onChange, setOpenItem, useAccordion, rowRefs]);

  const startDrag = (event: React.MouseEvent, index: number, openItem: string) => {
    if (!allowReorder || items.length < 2) return;
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();

    const wasOpen = openItem === String(index);
    if (wasOpen) {
      setOpenItem("");
    }

    setDragState({
      items: [...items],
      activeIndex: index,
      openAtStart: wasOpen,
    });
  };

  return { dragState, renderedItems, startDrag };
}
