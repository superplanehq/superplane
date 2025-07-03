var $01d3f539e91688c8$exports = require("./context.main.js");
var $3maub$reactariautils = require("@react-aria/utils");
var $3maub$react = require("react");


function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "PressResponder", () => $3596bae48579386f$export$3351871ee4b288b8);
$parcel$export(module.exports, "ClearPressResponder", () => $3596bae48579386f$export$cf75428e0b9ed1ea);
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


const $3596bae48579386f$export$3351871ee4b288b8 = /*#__PURE__*/ (0, ($parcel$interopDefault($3maub$react))).forwardRef(({ children: children, ...props }, ref)=>{
    let isRegistered = (0, $3maub$react.useRef)(false);
    let prevContext = (0, $3maub$react.useContext)((0, $01d3f539e91688c8$exports.PressResponderContext));
    ref = (0, $3maub$reactariautils.useObjectRef)(ref || (prevContext === null || prevContext === void 0 ? void 0 : prevContext.ref));
    let context = (0, $3maub$reactariautils.mergeProps)(prevContext || {}, {
        ...props,
        ref: ref,
        register () {
            isRegistered.current = true;
            if (prevContext) prevContext.register();
        }
    });
    (0, $3maub$reactariautils.useSyncRef)(prevContext, ref);
    (0, $3maub$react.useEffect)(()=>{
        if (!isRegistered.current) {
            if (process.env.NODE_ENV !== 'production') console.warn("A PressResponder was rendered without a pressable child. Either call the usePress hook, or wrap your DOM node with <Pressable> component.");
            isRegistered.current = true; // only warn once in strict mode.
        }
    }, []);
    return /*#__PURE__*/ (0, ($parcel$interopDefault($3maub$react))).createElement((0, $01d3f539e91688c8$exports.PressResponderContext).Provider, {
        value: context
    }, children);
});
function $3596bae48579386f$export$cf75428e0b9ed1ea({ children: children }) {
    let context = (0, $3maub$react.useMemo)(()=>({
            register: ()=>{}
        }), []);
    return /*#__PURE__*/ (0, ($parcel$interopDefault($3maub$react))).createElement((0, $01d3f539e91688c8$exports.PressResponderContext).Provider, {
        value: context
    }, children);
}


//# sourceMappingURL=PressResponder.main.js.map
