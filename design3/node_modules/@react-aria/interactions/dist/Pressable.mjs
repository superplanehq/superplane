import {usePress as $f6c31cce2adf654f$export$45712eceda6fad21} from "./usePress.mjs";
import {useFocusable as $f645667febf57a63$export$4c014de7c8940b4c} from "./useFocusable.mjs";
import {useObjectRef as $hhDyF$useObjectRef, getOwnerWindow as $hhDyF$getOwnerWindow, isFocusable as $hhDyF$isFocusable, mergeProps as $hhDyF$mergeProps, mergeRefs as $hhDyF$mergeRefs} from "@react-aria/utils";
import $hhDyF$react, {useEffect as $hhDyF$useEffect} from "react";

/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 



const $3b117e43dc0ca95d$export$27c701ed9e449e99 = /*#__PURE__*/ (0, $hhDyF$react).forwardRef(({ children: children, ...props }, ref)=>{
    ref = (0, $hhDyF$useObjectRef)(ref);
    let { pressProps: pressProps } = (0, $f6c31cce2adf654f$export$45712eceda6fad21)({
        ...props,
        ref: ref
    });
    let { focusableProps: focusableProps } = (0, $f645667febf57a63$export$4c014de7c8940b4c)(props, ref);
    let child = (0, $hhDyF$react).Children.only(children);
    (0, $hhDyF$useEffect)(()=>{
        if (process.env.NODE_ENV === 'production') return;
        let el = ref.current;
        if (!el || !(el instanceof (0, $hhDyF$getOwnerWindow)(el).Element)) {
            console.error('<Pressable> child must forward its ref to a DOM element.');
            return;
        }
        if (!props.isDisabled && !(0, $hhDyF$isFocusable)(el)) {
            console.warn('<Pressable> child must be focusable. Please ensure the tabIndex prop is passed through.');
            return;
        }
        if (el.localName !== 'button' && el.localName !== 'input' && el.localName !== 'select' && el.localName !== 'textarea' && el.localName !== 'a' && el.localName !== 'area' && el.localName !== 'summary') {
            let role = el.getAttribute('role');
            if (!role) console.warn('<Pressable> child must have an interactive ARIA role.');
            else if (// https://w3c.github.io/aria/#widget_roles
            role !== 'application' && role !== 'button' && role !== 'checkbox' && role !== 'combobox' && role !== 'gridcell' && role !== 'link' && role !== 'menuitem' && role !== 'menuitemcheckbox' && role !== 'menuitemradio' && role !== 'option' && role !== 'radio' && role !== 'searchbox' && role !== 'separator' && role !== 'slider' && role !== 'spinbutton' && role !== 'switch' && role !== 'tab' && role !== 'textbox' && role !== 'treeitem') console.warn(`<Pressable> child must have an interactive ARIA role. Got "${role}".`);
        }
    }, [
        ref,
        props.isDisabled
    ]);
    // @ts-ignore
    let childRef = parseInt((0, $hhDyF$react).version, 10) < 19 ? child.ref : child.props.ref;
    return /*#__PURE__*/ (0, $hhDyF$react).cloneElement(child, {
        ...(0, $hhDyF$mergeProps)(pressProps, focusableProps, child.props),
        // @ts-ignore
        ref: (0, $hhDyF$mergeRefs)(childRef, ref)
    });
});


export {$3b117e43dc0ca95d$export$27c701ed9e449e99 as Pressable};
//# sourceMappingURL=Pressable.module.js.map
