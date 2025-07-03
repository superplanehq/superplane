/// <reference types="react" />
import { PopoverMachine } from './popover-machine.js';
export declare const PopoverContext: import("react").Context<PopoverMachine | null>;
export declare function usePopoverMachineContext(component: string): PopoverMachine;
export declare function usePopoverMachine({ id, __demoMode, }: {
    id: string;
    __demoMode?: boolean;
}): PopoverMachine;
