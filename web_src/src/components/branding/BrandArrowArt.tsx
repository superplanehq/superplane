import { useContext, useEffect, useState } from "react";
import field from "@/assets/superplane-auth-field.svg";
import { ThemeContext } from "@/contexts/themeContextState";
import { FirstRunArtPane } from "@/pages/factories/pages/onboarding/first-run/FirstRunArtPane";

/** Reuse the onboarding renderer, with static artwork for reduced motion. */
export function BrandArrowArt({
  variant = "orange",
  minWidth = 851,
}: {
  variant?: "orange" | "teal";
  minWidth?: number;
}) {
  const theme = useContext(ThemeContext);
  const [canAnimate, setCanAnimate] = useState(false);
  const dark = theme?.resolvedTheme === "dark";
  const background = variant === "orange" ? "#ed8e09" : dark ? "#0b383a" : "#87ae9d";
  const arrowColor = variant === "teal" && dark ? "#f3f3ea" : "#08090a";

  useEffect(() => {
    const media = window.matchMedia(`(min-width: ${minWidth}px) and (prefers-reduced-motion: no-preference)`);
    const sync = () => setCanAnimate(media.matches);
    sync();
    media.addEventListener("change", sync);
    return () => media.removeEventListener("change", sync);
  }, [minWidth]);

  return (
    <div className="brand-arrow-art" style={{ background }} aria-hidden="true">
      <img src={field} alt="" className="brand-arrow-fallback" />
      {canAnimate ? <FirstRunArtPane art={{ mode: "school", background, arrowColor }} /> : null}
    </div>
  );
}
