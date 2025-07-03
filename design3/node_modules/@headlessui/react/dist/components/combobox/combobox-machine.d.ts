import { Machine } from '../../machine.js';
import type { EnsureArray } from '../../types.js';
import { Focus } from '../../utils/calculate-active-index.js';
interface MutableRefObject<T> {
    current: T;
}
export declare enum ComboboxState {
    Open = 0,
    Closed = 1
}
export declare enum ValueMode {
    Single = 0,
    Multi = 1
}
export declare enum ActivationTrigger {
    Pointer = 0,
    Focus = 1,
    Other = 2
}
export type ComboboxOptionDataRef<T> = MutableRefObject<{
    disabled: boolean;
    value: T;
    domRef: MutableRefObject<HTMLElement | null>;
    order: number | null;
}>;
export interface State<T> {
    id: string;
    dataRef: MutableRefObject<{
        value: unknown;
        defaultValue: unknown;
        disabled: boolean;
        invalid: boolean;
        mode: ValueMode;
        immediate: boolean;
        onChange: (value: T) => void;
        onClose?: () => void;
        compare(a: unknown, z: unknown): boolean;
        isSelected(value: unknown): boolean;
        virtual: {
            options: T[];
            disabled: (value: T) => boolean;
        } | null;
        calculateIndex(value: unknown): number;
        __demoMode: boolean;
        optionsPropsRef: MutableRefObject<{
            static: boolean;
            hold: boolean;
        }>;
    }>;
    virtual: {
        options: T[];
        disabled: (value: unknown) => boolean;
    } | null;
    comboboxState: ComboboxState;
    defaultToFirstOption: boolean;
    options: {
        id: string;
        dataRef: ComboboxOptionDataRef<T>;
    }[];
    activeOptionIndex: number | null;
    activationTrigger: ActivationTrigger;
    isTyping: boolean;
    inputElement: HTMLInputElement | null;
    buttonElement: HTMLButtonElement | null;
    optionsElement: HTMLElement | null;
    __demoMode: boolean;
}
export declare enum ActionTypes {
    OpenCombobox = 0,
    CloseCombobox = 1,
    GoToOption = 2,
    SetTyping = 3,
    RegisterOption = 4,
    UnregisterOption = 5,
    DefaultToFirstOption = 6,
    SetActivationTrigger = 7,
    UpdateVirtualConfiguration = 8,
    SetInputElement = 9,
    SetButtonElement = 10,
    SetOptionsElement = 11
}
type Actions<T> = {
    type: ActionTypes.CloseCombobox;
} | {
    type: ActionTypes.OpenCombobox;
} | {
    type: ActionTypes.GoToOption;
    focus: Focus.Specific;
    idx: number;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.SetTyping;
    isTyping: boolean;
} | {
    type: ActionTypes.GoToOption;
    focus: Exclude<Focus, Focus.Specific>;
    trigger?: ActivationTrigger;
} | {
    type: ActionTypes.RegisterOption;
    payload: {
        id: string;
        dataRef: ComboboxOptionDataRef<T>;
    };
} | {
    type: ActionTypes.UnregisterOption;
    id: string;
} | {
    type: ActionTypes.DefaultToFirstOption;
    value: boolean;
} | {
    type: ActionTypes.SetActivationTrigger;
    trigger: ActivationTrigger;
} | {
    type: ActionTypes.UpdateVirtualConfiguration;
    options: T[];
    disabled: ((value: any) => boolean) | null;
} | {
    type: ActionTypes.SetInputElement;
    element: HTMLInputElement | null;
} | {
    type: ActionTypes.SetButtonElement;
    element: HTMLButtonElement | null;
} | {
    type: ActionTypes.SetOptionsElement;
    element: HTMLElement | null;
};
export declare class ComboboxMachine<T> extends Machine<State<T>, Actions<T>> {
    static new<T, TMultiple extends boolean | undefined>({ id, virtual, __demoMode, }: {
        id: string;
        virtual?: {
            options: TMultiple extends true ? EnsureArray<NoInfer<T>> : NoInfer<T>[];
            disabled?: (value: TMultiple extends true ? EnsureArray<NoInfer<T>>[number] : NoInfer<T>) => boolean;
        } | null;
        __demoMode?: boolean;
    }): ComboboxMachine<any>;
    constructor(initialState: State<T>);
    actions: {
        onChange: (newValue: T) => void;
        registerOption: (id: string, dataRef: ComboboxOptionDataRef<T>) => () => void;
        goToOption: (focus: {
            focus: Focus.Specific;
            idx: number;
        } | {
            focus: Exclude<Focus, Focus.Specific>;
        }, trigger?: ActivationTrigger) => void;
        setIsTyping: (isTyping: boolean) => void;
        closeCombobox: () => void;
        openCombobox: () => void;
        setActivationTrigger: (trigger: ActivationTrigger) => void;
        selectActiveOption: () => void;
        setInputElement: (element: HTMLInputElement | null) => void;
        setButtonElement: (element: HTMLButtonElement | null) => void;
        setOptionsElement: (element: HTMLElement | null) => void;
    };
    selectors: {
        activeDescendantId: (state: State<T>) => string | undefined;
        activeOptionIndex: (state: State<T>) => number | null;
        activeOption: (state: State<T>) => T | null;
        isActive: (state: State<T>, value: T, id: string) => boolean;
        shouldScrollIntoView: (state: State<T>, value: T, id: string) => boolean;
    };
    reduce(state: Readonly<State<T>>, action: Actions<T>): State<T>;
}
export {};
