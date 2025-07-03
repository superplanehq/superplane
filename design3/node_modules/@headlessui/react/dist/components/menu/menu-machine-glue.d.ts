/// <reference types="react" />
import { MenuMachine } from './menu-machine.js';
export declare const MenuContext: import("react").Context<MenuMachine | null>;
export declare function useMenuMachineContext(component: string): MenuMachine;
export declare function useMenuMachine({ id, __demoMode }: {
    id: string;
    __demoMode?: boolean;
}): MenuMachine;
