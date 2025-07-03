/// <reference types="react" />
import { ListboxMachine } from './listbox-machine.js';
export declare const ListboxContext: import("react").Context<ListboxMachine<unknown> | null>;
export declare function useListboxMachineContext<T>(component: string): ListboxMachine<T>;
export declare function useListboxMachine({ id, __demoMode, }: {
    id: string;
    __demoMode?: boolean;
}): ListboxMachine<unknown>;
