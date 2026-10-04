import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useIsMobile } from "@/hooks/use-mobile";
import { FEATURE_MOBILE_FACTORY_BOARD } from "@/lib/experimentalFeatures";

/**
 * True when the workspace should render the mobile shell: the organization
 * has the mobile board feature and the viewport is phone width. Desktop
 * screens always keep the standard shell, even with the feature on.
 */
export function useMobileFactoryShell(organizationId?: string): boolean {
  const isMobile = useIsMobile();
  const { has } = useExperimentalFeature(organizationId);
  return isMobile && has(FEATURE_MOBILE_FACTORY_BOARD);
}
