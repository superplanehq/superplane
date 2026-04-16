import type { HTMLAttributes, SyntheticEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

type ContextMenuPosition = {
  x: number;
  y: number;
};

type ContextMenuState<TData> = {
  data: TData;
  position: ContextMenuPosition;
};

export function useContextMenu<TData>() {
  const [menuState, setMenuState] = useState<ContextMenuState<TData> | null>(null);
  const [menuPosition, setMenuPosition] = useState<ContextMenuPosition | null>(null);
  const [selectedGroupKey, setSelectedGroupKey] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const closeContextMenu = useCallback(() => {
    setMenuState(null);
    setMenuPosition(null);
    setSelectedGroupKey(null);
  }, []);

  const openContextMenu = useCallback((position: ContextMenuPosition, data: TData) => {
    setMenuState({ position, data });
    setMenuPosition(position);
    setSelectedGroupKey(null);
  }, []);

  const stopInteraction = useCallback((event: SyntheticEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
  }, []);

  const backdropProps = useMemo<HTMLAttributes<HTMLDivElement>>(
    () => ({
      onMouseDown: (event) => {
        event.preventDefault();
        event.stopPropagation();
        closeContextMenu();
      },
      onPointerDown: (event) => {
        event.preventDefault();
        event.stopPropagation();
        closeContextMenu();
      },
      onTouchStart: (event) => {
        event.preventDefault();
        event.stopPropagation();
        closeContextMenu();
      },
      onWheelCapture: (event) => {
        stopInteraction(event);
      },
      onTouchMoveCapture: (event) => {
        stopInteraction(event);
      },
      onContextMenu: (event) => {
        stopInteraction(event);
      },
    }),
    [closeContextMenu, stopInteraction],
  );

  const menuProps = useMemo<HTMLAttributes<HTMLDivElement>>(
    () => ({
      onMouseDown: (event) => {
        event.stopPropagation();
      },
      onPointerDown: (event) => {
        event.stopPropagation();
      },
      onWheelCapture: (event) => {
        stopInteraction(event);
      },
      onTouchMoveCapture: (event) => {
        stopInteraction(event);
      },
      onContextMenu: (event) => {
        stopInteraction(event);
      },
    }),
    [stopInteraction],
  );

  useEffect(() => {
    if (!menuState) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (menuRef.current?.contains(event.target as Node)) {
        return;
      }

      closeContextMenu();
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closeContextMenu();
      }
    };

    const handleResize = () => {
      closeContextMenu();
    };

    document.addEventListener("mousedown", handlePointerDown, true);
    document.addEventListener("keydown", handleKeyDown);
    window.addEventListener("resize", handleResize);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown, true);
      document.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("resize", handleResize);
    };
  }, [closeContextMenu, menuState]);

  useEffect(() => {
    if (!menuState || !menuRef.current) {
      return;
    }

    const rect = menuRef.current.getBoundingClientRect();
    const nextX = Math.min(menuState.position.x, Math.max(8, window.innerWidth - rect.width - 8));
    const nextY = Math.min(menuState.position.y, Math.max(8, window.innerHeight - rect.height - 8));

    if (nextX !== menuPosition?.x || nextY !== menuPosition?.y) {
      setMenuPosition({ x: nextX, y: nextY });
    }
  }, [menuPosition?.x, menuPosition?.y, menuState]);

  return {
    contextMenuData: menuState?.data ?? null,
    contextMenuPosition: menuPosition,
    isContextMenuOpen: Boolean(menuState && menuPosition),
    menuRef,
    selectedGroupKey,
    setSelectedGroupKey,
    backdropProps,
    menuProps,
    openContextMenu,
    closeContextMenu,
  };
}
