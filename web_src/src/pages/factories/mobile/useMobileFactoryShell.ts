import { useIsMobile } from "@/hooks/use-mobile";

/**
 * True when the workspace should render the mobile shell: the viewport is
 * phone width. Desktop screens always keep the standard shell.
 */
export function useMobileFactoryShell(): boolean {
  return useIsMobile();
}
