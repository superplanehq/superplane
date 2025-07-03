import { type MouseEventHandler } from 'react';
import { Machine } from '../../machine.js';
type MouseEvent<T> = Parameters<MouseEventHandler<T>>[0];
export declare enum PopoverStates {
    Open = 0,
    Closed = 1
}
interface State {
    id: string;
    popoverState: PopoverStates;
    buttons: {
        current: Symbol[];
    };
    button: HTMLElement | null;
    buttonId: string | null;
    panel: HTMLElement | null;
    panelId: string | null;
    beforePanelSentinel: {
        current: HTMLButtonElement | null;
    };
    afterPanelSentinel: {
        current: HTMLButtonElement | null;
    };
    afterButtonSentinel: {
        current: HTMLButtonElement | null;
    };
    __demoMode: boolean;
}
export declare enum ActionTypes {
    OpenPopover = 0,
    ClosePopover = 1,
    SetButton = 2,
    SetButtonId = 3,
    SetPanel = 4,
    SetPanelId = 5
}
export type Actions = {
    type: ActionTypes.OpenPopover;
} | {
    type: ActionTypes.ClosePopover;
} | {
    type: ActionTypes.SetButton;
    button: HTMLElement | null;
} | {
    type: ActionTypes.SetButtonId;
    buttonId: string | null;
} | {
    type: ActionTypes.SetPanel;
    panel: HTMLElement | null;
} | {
    type: ActionTypes.SetPanelId;
    panelId: string | null;
};
export declare class PopoverMachine extends Machine<State, Actions> {
    static new({ id, __demoMode }: {
        id: string;
        __demoMode?: boolean;
    }): PopoverMachine;
    constructor(initialState: State);
    reduce(state: Readonly<State>, action: Actions): State;
    actions: {
        close: () => void;
        refocusableClose: (focusableElement?: HTMLElement | {
            current: HTMLElement | null;
        } | MouseEvent<HTMLElement>) => void;
        open: () => void;
        setButtonId: (id: string | null) => void;
        setButton: (button: HTMLElement | null) => void;
        setPanelId: (id: string | null) => void;
        setPanel: (panel: HTMLElement | null) => void;
    };
    selectors: {
        isPortalled: (state: State) => boolean;
    };
}
export {};
