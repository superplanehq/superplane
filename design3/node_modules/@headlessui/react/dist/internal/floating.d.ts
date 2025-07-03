import { type InnerProps } from '@floating-ui/react';
import * as React from 'react';
type Align = 'start' | 'end';
type Placement = 'top' | 'right' | 'bottom' | 'left';
type AnchorTo = `${Placement}` | `${Placement} ${Align}`;
type AnchorToWithSelection = `${Placement | 'selection'}` | `${Placement | 'selection'} ${Align}`;
type BaseAnchorProps = {
    /**
     * The `gap` is the space between the trigger and the panel.
     */
    gap: number | string;
    /**
     * The `offset` is the amount the panel should be nudged from its original position.
     */
    offset: number | string;
    /**
     * The `padding` is the minimum space between the panel and the viewport.
     */
    padding: number | string;
};
export type AnchorProps = false | AnchorTo | Partial<BaseAnchorProps & {
    /**
     * The `to` value defines which side of the trigger the panel should be placed on and its
     * alignment.
     */
    to: AnchorTo;
}>;
export type AnchorPropsWithSelection = false | AnchorToWithSelection | Partial<BaseAnchorProps & {
    /**
     * The `to` value defines which side of the trigger the panel should be placed on and its
     * alignment.
     */
    to: AnchorToWithSelection;
}>;
export type InternalFloatingPanelProps = Partial<{
    inner: {
        listRef: InnerProps['listRef'];
        index: InnerProps['index'];
    };
}>;
export declare function useResolvedAnchor<T extends AnchorProps | AnchorPropsWithSelection>(anchor?: T): Exclude<T, boolean | string> | null;
export declare function useFloatingReference(): ((node: import("@floating-ui/react-dom").ReferenceType | null) => void) & ((node: any) => void);
export declare function useFloatingReferenceProps(): (userProps?: React.HTMLProps<Element> | undefined) => Record<string, unknown>;
export declare function useFloatingPanelProps(): (userProps?: React.HTMLProps<HTMLElement> | undefined) => Record<string, unknown> & {
    'data-anchor': AnchorToWithSelection | undefined;
};
export declare function useFloatingPanel(placement?: (AnchorPropsWithSelection & InternalFloatingPanelProps) | null): readonly [((node: HTMLElement | null) => void) & ((node: HTMLElement | null) => void), React.CSSProperties | undefined];
export declare function FloatingProvider({ children, enabled, }: {
    children: React.ReactNode;
    enabled?: boolean;
}): React.JSX.Element;
export {};
