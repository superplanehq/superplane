import { Machine } from '../../machine.js';
import { Focus } from '../../utils/calculate-active-index.js';
interface MutableRefObject<T> {
    current: T;
}
export declare enum ListboxStates {
    Open = 0,
    Closed = 1
}
export declare enum ValueMode {
    Single = 0,
    Multi = 1
}
export declare enum ActivationTrigger {
    Pointer = 0,
    Other = 1
}
type ListboxOptionDataRef<T> = MutableRefObject<{
    textValue?: string;
    disabled: boolean;
    value: T;
    domRef: MutableRefObject<HTMLElement | null>;
}>;
interface State<T> {
    id: string;
    __demoMode: boolean;
    dataRef: MutableRefObject<{
        value: unknown;
        disabled: boolean;
        invalid: boolean;
        mode: ValueMode;
        orientation: 'horizontal' | 'vertical';
        onChange: (value: T) => void;
        compare(a: unknown, z: unknown): boolean;
        isSelected(value: unknown): boolean;
        optionsPropsRef: MutableRefObject<{
            static: boolean;
            hold: boolean;
        }>;
        listRef: MutableRefObject<Map<string, HTMLElement | null>>;
    }>;
    listboxState: ListboxStates;
    options: {
        id: string;
        dataRef: ListboxOptionDataRef<T>;
    }[];
    searchQuery: string;
    activeOptionIndex: number | null;
    activationTrigger: ActivationTrigger;
    buttonElement: HTMLButtonElement | null;
    optionsElement: HTMLElement | null;
    pendingShouldSort: boolean;
    pendingFocus: {
        focus: Exclude<Focus, Focus.Specific>;
    } | {
        focus: Focus.Specific;
        id: string;
    };
}
export declare enum ActionTypes {
    OpenListbox = 0,
    CloseListbox = 1,
    GoToOption = 2,
    Search = 3,
    ClearSearch = 4,
    RegisterOptions = 5,
    UnregisterOptions = 6,
    SetButtonElement = 7,
    SetOptionsElement = 8,
    SortOptions = 9
}
type Actions<T> = {
    type: ActionTypes.CloseListbox;
} | {
    type: ActionTypes.OpenListbox;
    focus: {
        focus: Exclude<Focus, Focus.Specific>;
    } | {
        focus: Focus.Specific;
        id: string;
    };
} | {
    type: ActionTypes.GoToOption;
    focus: Focus.Specific;
    id: string;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.GoToOption;
    focus: Exclude<Focus, Focus.Specific>;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.Search;
    value: string;
} | {
    type: ActionTypes.ClearSearch;
} | {
    type: ActionTypes.RegisterOptions;
    options: {
        id: string;
        dataRef: ListboxOptionDataRef<T>;
    }[];
} | {
    type: ActionTypes.UnregisterOptions;
    options: string[];
} | {
    type: ActionTypes.SetButtonElement;
    element: HTMLButtonElement | null;
} | {
    type: ActionTypes.SetOptionsElement;
    element: HTMLElement | null;
} | {
    type: ActionTypes.SortOptions;
};
export declare class ListboxMachine<T> extends Machine<State<T>, Actions<T>> {
    static new({ id, __demoMode }: {
        id: string;
        __demoMode?: boolean;
    }): ListboxMachine<unknown>;
    constructor(initialState: State<T>);
    actions: {
        onChange: (newValue: T) => void;
        registerOption: (id: string, dataRef: ListboxOptionDataRef<T>) => void;
        unregisterOption: (id: string) => void;
        goToOption: (focus: {
            focus: Focus.Specific;
            id: string;
        } | {
            focus: Exclude<Focus, Focus.Specific>;
        }, trigger?: ActivationTrigger | undefined) => void;
        closeListbox: () => void;
        openListbox: (focus: {
            focus: Exclude<Focus, Focus.Specific>;
        } | {
            focus: Focus.Specific;
            id: string;
        }) => void;
        selectActiveOption: () => void;
        selectOption: (id: string) => void;
        search: (value: string) => void;
        clearSearch: () => void;
        setButtonElement: (element: HTMLButtonElement | null) => void;
        setOptionsElement: (element: HTMLElement | null) => void;
    };
    selectors: {
        activeDescendantId(state: State<T>): string | undefined;
        isActive(state: State<T>, id: string): boolean;
        shouldScrollIntoView(state: State<T>, id: string): boolean;
    };
    reduce(state: Readonly<State<T>>, action: Actions<T>): State<T>;
}
export {};
