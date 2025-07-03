import { JSX, ReactNode } from "react";
export interface SSRProviderProps {
    /** Your application here. */
    children: ReactNode;
}
/**
 * When using SSR with React Aria in React 16 or 17, applications must be wrapped in an SSRProvider.
 * This ensures that auto generated ids are consistent between the client and server.
 */
export function SSRProvider(props: SSRProviderProps): JSX.Element;
declare function useModernSSRSafeId(defaultId?: string): string;
/** @private */
export const useSSRSafeId: typeof useModernSSRSafeId;
/**
 * Returns whether the component is currently being server side rendered or
 * hydrated on the client. Can be used to delay browser-specific rendering
 * until after hydration.
 */
export function useIsSSR(): boolean;

//# sourceMappingURL=types.d.ts.map
