import * as React from "react";

const MOBILE_BREAKPOINT = 768;

function viewportIsMobile(): boolean {
  return typeof window !== "undefined" && window.innerWidth < MOBILE_BREAKPOINT;
}

/**
 * True below the phone breakpoint. Reads the viewport during the first render
 * so route-level switches (redirects, shells) pick the right branch at once
 * instead of rendering the desktop branch for one frame.
 */
export function useIsMobile() {
  const [isMobile, setIsMobile] = React.useState<boolean>(viewportIsMobile);

  React.useEffect(() => {
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    const onChange = () => {
      setIsMobile(viewportIsMobile());
    };
    mql.addEventListener("change", onChange);
    setIsMobile(viewportIsMobile());
    return () => mql.removeEventListener("change", onChange);
  }, []);

  return isMobile;
}
