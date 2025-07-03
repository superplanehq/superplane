declare enum ActionKind {
    Ignore = 0,
    Select = 1,
    Close = 2
}
export declare const Action: {
    /** Do nothing */
    Ignore: {
        readonly kind: ActionKind.Ignore;
    };
    /** Select the current item */
    Select: (target: HTMLElement) => {
        readonly kind: ActionKind.Select;
        readonly target: HTMLElement;
    };
    /** Close the dropdown */
    Close: {
        readonly kind: ActionKind.Close;
    };
};
type PointerEventWithTarget = Exclude<PointerEvent, 'target'> & {
    target: HTMLElement;
};
export declare function useQuickRelease(enabled: boolean, { trigger, action, close, select, }: {
    trigger: HTMLElement | null;
    action: (e: PointerEventWithTarget) => {
        kind: ActionKind.Ignore;
    } | {
        kind: ActionKind.Select;
        target: HTMLElement;
    } | {
        kind: ActionKind.Close;
    };
    close: () => void;
    select: (target: HTMLElement) => void;
}): void;
export {};
