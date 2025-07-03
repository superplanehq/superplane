import { Machine } from '../../machine.js';
import { Focus } from '../../utils/calculate-active-index.js';
export declare enum MenuState {
    Open = 0,
    Closed = 1
}
export declare enum ActivationTrigger {
    Pointer = 0,
    Other = 1
}
export type MenuItemDataRef = {
    current: {
        textValue?: string;
        disabled: boolean;
        domRef: {
            current: HTMLElement | null;
        };
    };
};
export interface State {
    id: string;
    __demoMode: boolean;
    menuState: MenuState;
    buttonElement: HTMLButtonElement | null;
    itemsElement: HTMLElement | null;
    items: {
        id: string;
        dataRef: MenuItemDataRef;
    }[];
    searchQuery: string;
    activeItemIndex: number | null;
    activationTrigger: ActivationTrigger;
    pendingShouldSort: boolean;
    pendingFocus: {
        focus: Exclude<Focus, Focus.Specific>;
    } | {
        focus: Focus.Specific;
        id: string;
    };
}
export declare enum ActionTypes {
    OpenMenu = 0,
    CloseMenu = 1,
    GoToItem = 2,
    Search = 3,
    ClearSearch = 4,
    RegisterItems = 5,
    UnregisterItems = 6,
    SetButtonElement = 7,
    SetItemsElement = 8,
    SortItems = 9
}
export type Actions = {
    type: ActionTypes.CloseMenu;
} | {
    type: ActionTypes.OpenMenu;
    focus: {
        focus: Exclude<Focus, Focus.Specific>;
    } | {
        focus: Focus.Specific;
        id: string;
    };
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.GoToItem;
    focus: Focus.Specific;
    id: string;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.GoToItem;
    focus: Exclude<Focus, Focus.Specific>;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.Search;
    value: string;
} | {
    type: ActionTypes.ClearSearch;
} | {
    type: ActionTypes.RegisterItems;
    items: {
        id: string;
        dataRef: MenuItemDataRef;
    }[];
} | {
    type: ActionTypes.UnregisterItems;
    items: string[];
} | {
    type: ActionTypes.SetButtonElement;
    element: HTMLButtonElement | null;
} | {
    type: ActionTypes.SetItemsElement;
    element: HTMLElement | null;
} | {
    type: ActionTypes.SortItems;
};
export declare class MenuMachine extends Machine<State, Actions> {
    static new({ id, __demoMode }: {
        id: string;
        __demoMode?: boolean;
    }): MenuMachine;
    constructor(initialState: State);
    reduce(state: Readonly<State>, action: Actions): State;
    actions: {
        registerItem: (id: string, dataRef: MenuItemDataRef) => void;
        unregisterItem: (id: string) => void;
    };
    selectors: {
        activeDescendantId(state: State): string | undefined;
        isActive(state: State, id: string): boolean;
        shouldScrollIntoView(state: State, id: string): boolean;
    };
}
