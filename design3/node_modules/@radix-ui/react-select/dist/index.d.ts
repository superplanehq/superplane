import * as React from "react";
import { DismissableLayer } from "@radix-ui/react-dismissable-layer";
import { FocusScope } from "@radix-ui/react-focus-scope";
import * as PopperPrimitive from "@radix-ui/react-popper";
import { Portal as _Portal1 } from "@radix-ui/react-portal";
import * as Radix from "@radix-ui/react-primitive";
import { Primitive } from "@radix-ui/react-primitive";
type Direction = 'ltr' | 'rtl';
export const createSelectScope: import("@radix-ui/react-context").CreateScope;
export interface SelectProps {
    children?: React.ReactNode;
    value?: string;
    defaultValue?: string;
    onValueChange?(value: string): void;
    open?: boolean;
    defaultOpen?: boolean;
    onOpenChange?(open: boolean): void;
    dir?: Direction;
    name?: string;
    autoComplete?: string;
    disabled?: boolean;
    required?: boolean;
}
export const Select: React.FC<SelectProps>;
type PrimitiveButtonProps = Radix.ComponentPropsWithoutRef<typeof Primitive.button>;
export interface SelectTriggerProps extends PrimitiveButtonProps {
}
export const SelectTrigger: React.ForwardRefExoticComponent<SelectTriggerProps & React.RefAttributes<HTMLButtonElement>>;
type PrimitiveSpanProps = Radix.ComponentPropsWithoutRef<typeof Primitive.span>;
export interface SelectValueProps extends Omit<PrimitiveSpanProps, 'placeholder'> {
    placeholder?: React.ReactNode;
}
export const SelectValue: React.ForwardRefExoticComponent<SelectValueProps & React.RefAttributes<HTMLSpanElement>>;
export interface SelectIconProps extends PrimitiveSpanProps {
}
export const SelectIcon: React.ForwardRefExoticComponent<SelectIconProps & React.RefAttributes<HTMLSpanElement>>;
type PortalProps = React.ComponentPropsWithoutRef<typeof _Portal1>;
export interface SelectPortalProps extends Omit<PortalProps, 'asChild'> {
    children?: React.ReactNode;
}
export const SelectPortal: React.FC<SelectPortalProps>;
export interface SelectContentProps extends SelectContentImplProps {
}
export const SelectContent: React.ForwardRefExoticComponent<SelectContentProps & React.RefAttributes<HTMLDivElement>>;
type DismissableLayerProps = React.ComponentPropsWithoutRef<typeof DismissableLayer>;
type FocusScopeProps = Radix.ComponentPropsWithoutRef<typeof FocusScope>;
type SelectPopperPrivateProps = {
    onPlaced?: PopperContentProps['onPlaced'];
};
interface SelectContentImplProps extends Omit<SelectPopperPositionProps, keyof SelectPopperPrivateProps>, Omit<SelectItemAlignedPositionProps, keyof SelectPopperPrivateProps> {
    /**
     * Event handler called when auto-focusing on close.
     * Can be prevented.
     */
    onCloseAutoFocus?: FocusScopeProps['onUnmountAutoFocus'];
    /**
     * Event handler called when the escape key is down.
     * Can be prevented.
     */
    onEscapeKeyDown?: DismissableLayerProps['onEscapeKeyDown'];
    /**
     * Event handler called when the a `pointerdown` event happens outside of the `DismissableLayer`.
     * Can be prevented.
     */
    onPointerDownOutside?: DismissableLayerProps['onPointerDownOutside'];
    position?: 'item-aligned' | 'popper';
}
interface SelectItemAlignedPositionProps extends PrimitiveDivProps, SelectPopperPrivateProps {
}
type PopperContentProps = React.ComponentPropsWithoutRef<typeof PopperPrimitive.Content>;
interface SelectPopperPositionProps extends PopperContentProps, SelectPopperPrivateProps {
}
type PrimitiveDivProps = Radix.ComponentPropsWithoutRef<typeof Primitive.div>;
export interface SelectViewportProps extends PrimitiveDivProps {
}
export const SelectViewport: React.ForwardRefExoticComponent<SelectViewportProps & React.RefAttributes<HTMLDivElement>>;
export interface SelectGroupProps extends PrimitiveDivProps {
}
export const SelectGroup: React.ForwardRefExoticComponent<SelectGroupProps & React.RefAttributes<HTMLDivElement>>;
export interface SelectLabelProps extends PrimitiveDivProps {
}
export const SelectLabel: React.ForwardRefExoticComponent<SelectLabelProps & React.RefAttributes<HTMLDivElement>>;
export interface SelectItemProps extends PrimitiveDivProps {
    value: string;
    disabled?: boolean;
    textValue?: string;
}
export const SelectItem: React.ForwardRefExoticComponent<SelectItemProps & React.RefAttributes<HTMLDivElement>>;
export interface SelectItemTextProps extends PrimitiveSpanProps {
}
export const SelectItemText: React.ForwardRefExoticComponent<SelectItemTextProps & React.RefAttributes<HTMLSpanElement>>;
export interface SelectItemIndicatorProps extends PrimitiveSpanProps {
}
export const SelectItemIndicator: React.ForwardRefExoticComponent<SelectItemIndicatorProps & React.RefAttributes<HTMLSpanElement>>;
export interface SelectScrollUpButtonProps extends Omit<SelectScrollButtonImplProps, 'onAutoScroll'> {
}
export const SelectScrollUpButton: React.ForwardRefExoticComponent<SelectScrollUpButtonProps & React.RefAttributes<HTMLDivElement>>;
export interface SelectScrollDownButtonProps extends Omit<SelectScrollButtonImplProps, 'onAutoScroll'> {
}
export const SelectScrollDownButton: React.ForwardRefExoticComponent<SelectScrollDownButtonProps & React.RefAttributes<HTMLDivElement>>;
interface SelectScrollButtonImplProps extends PrimitiveDivProps {
    onAutoScroll(): void;
}
export interface SelectSeparatorProps extends PrimitiveDivProps {
}
export const SelectSeparator: React.ForwardRefExoticComponent<SelectSeparatorProps & React.RefAttributes<HTMLDivElement>>;
type PopperArrowProps = Radix.ComponentPropsWithoutRef<typeof PopperPrimitive.Arrow>;
export interface SelectArrowProps extends PopperArrowProps {
}
export const SelectArrow: React.ForwardRefExoticComponent<SelectArrowProps & React.RefAttributes<SVGSVGElement>>;
export const Root: React.FC<SelectProps>;
export const Trigger: React.ForwardRefExoticComponent<SelectTriggerProps & React.RefAttributes<HTMLButtonElement>>;
export const Value: React.ForwardRefExoticComponent<SelectValueProps & React.RefAttributes<HTMLSpanElement>>;
export const Icon: React.ForwardRefExoticComponent<SelectIconProps & React.RefAttributes<HTMLSpanElement>>;
export const Portal: React.FC<SelectPortalProps>;
export const Content: React.ForwardRefExoticComponent<SelectContentProps & React.RefAttributes<HTMLDivElement>>;
export const Viewport: React.ForwardRefExoticComponent<SelectViewportProps & React.RefAttributes<HTMLDivElement>>;
export const Group: React.ForwardRefExoticComponent<SelectGroupProps & React.RefAttributes<HTMLDivElement>>;
export const Label: React.ForwardRefExoticComponent<SelectLabelProps & React.RefAttributes<HTMLDivElement>>;
export const Item: React.ForwardRefExoticComponent<SelectItemProps & React.RefAttributes<HTMLDivElement>>;
export const ItemText: React.ForwardRefExoticComponent<SelectItemTextProps & React.RefAttributes<HTMLSpanElement>>;
export const ItemIndicator: React.ForwardRefExoticComponent<SelectItemIndicatorProps & React.RefAttributes<HTMLSpanElement>>;
export const ScrollUpButton: React.ForwardRefExoticComponent<SelectScrollUpButtonProps & React.RefAttributes<HTMLDivElement>>;
export const ScrollDownButton: React.ForwardRefExoticComponent<SelectScrollDownButtonProps & React.RefAttributes<HTMLDivElement>>;
export const Separator: React.ForwardRefExoticComponent<SelectSeparatorProps & React.RefAttributes<HTMLDivElement>>;
export const Arrow: React.ForwardRefExoticComponent<SelectArrowProps & React.RefAttributes<SVGSVGElement>>;

//# sourceMappingURL=index.d.ts.map
