var $0294ea432cd92340$exports = require("./usePress.main.js");
var $15f8fd80892557ff$exports = require("./useFocusable.main.js");
var $ev4bP$reactariautils = require("@react-aria/utils");
var $ev4bP$react = require("react");


function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "Pressable", () => $e1dbec26039c051d$export$27c701ed9e449e99);
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



const $e1dbec26039c051d$export$27c701ed9e449e99 = /*#__PURE__*/ (0, ($parcel$interopDefault($ev4bP$react))).forwardRef(({ children: children, ...props }, ref)=>{
    ref = (0, $ev4bP$reactariautils.useObjectRef)(ref);
    let { pressProps: pressProps } = (0, $0294ea432cd92340$exports.usePress)({
        ...props,
        ref: ref
    });
    let { focusableProps: focusableProps } = (0, $15f8fd80892557ff$exports.useFocusable)(props, ref);
    let child = (0, ($parcel$interopDefault($ev4bP$react))).Children.only(children);
    (0, $ev4bP$react.useEffect)(()=>{
        if (process.env.NODE_ENV === 'production') return;
        let el = ref.current;
        if (!el || !(el instanceof (0, $ev4bP$reactariautils.getOwnerWindow)(el).Element)) {
            console.error('<Pressable> child must forward its ref to a DOM element.');
            return;
        }
        if (!props.isDisabled && !(0, $ev4bP$reactariautils.isFocusable)(el)) {
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
    let childRef = parseInt((0, ($parcel$interopDefault($ev4bP$react))).version, 10) < 19 ? child.ref : child.props.ref;
    return /*#__PURE__*/ (0, ($parcel$interopDefault($ev4bP$react))).cloneElement(child, {
        ...(0, $ev4bP$reactariautils.mergeProps)(pressProps, focusableProps, child.props),
        // @ts-ignore
        ref: (0, $ev4bP$reactariautils.mergeRefs)(childRef, ref)
    });
});


//# sourceMappingURL=Pressable.main.js.map
