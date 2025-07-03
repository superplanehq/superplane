/// <reference types="react" />
import { ComboboxMachine } from './combobox-machine.js';
export declare const ComboboxContext: import("react").Context<ComboboxMachine<unknown> | null>;
export declare function useComboboxMachineContext<T>(component: string): ComboboxMachine<T>;
export declare function useComboboxMachine({ id, virtual, __demoMode, }: Parameters<typeof ComboboxMachine.new>[0]): ComboboxMachine<any>;
