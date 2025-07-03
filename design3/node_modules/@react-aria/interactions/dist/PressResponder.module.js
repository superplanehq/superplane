import {PressResponderContext as $ae1eeba8b9eafd08$export$5165eccb35aaadb5} from "./context.module.js";
import {useObjectRef as $87RPk$useObjectRef, mergeProps as $87RPk$mergeProps, useSyncRef as $87RPk$useSyncRef} from "@react-aria/utils";
import $87RPk$react, {useRef as $87RPk$useRef, useContext as $87RPk$useContext, useEffect as $87RPk$useEffect, useMemo as $87RPk$useMemo} from "react";

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


const $f1ab8c75478c6f73$export$3351871ee4b288b8 = /*#__PURE__*/ (0, $87RPk$react).forwardRef(({ children: children, ...props }, ref)=>{
    let isRegistered = (0, $87RPk$useRef)(false);
    let prevContext = (0, $87RPk$useContext)((0, $ae1eeba8b9eafd08$export$5165eccb35aaadb5));
    ref = (0, $87RPk$useObjectRef)(ref || (prevContext === null || prevContext === void 0 ? void 0 : prevContext.ref));
    let context = (0, $87RPk$mergeProps)(prevContext || {}, {
        ...props,
        ref: ref,
        register () {
            isRegistered.current = true;
            if (prevContext) prevContext.register();
        }
    });
    (0, $87RPk$useSyncRef)(prevContext, ref);
    (0, $87RPk$useEffect)(()=>{
        if (!isRegistered.current) {
            if (process.env.NODE_ENV !== 'production') console.warn("A PressResponder was rendered without a pressable child. Either call the usePress hook, or wrap your DOM node with <Pressable> component.");
            isRegistered.current = true; // only warn once in strict mode.
        }
    }, []);
    return /*#__PURE__*/ (0, $87RPk$react).createElement((0, $ae1eeba8b9eafd08$export$5165eccb35aaadb5).Provider, {
        value: context
    }, children);
});
function $f1ab8c75478c6f73$export$cf75428e0b9ed1ea({ children: children }) {
    let context = (0, $87RPk$useMemo)(()=>({
            register: ()=>{}
        }), []);
    return /*#__PURE__*/ (0, $87RPk$react).createElement((0, $ae1eeba8b9eafd08$export$5165eccb35aaadb5).Provider, {
        value: context
    }, children);
}


export {$f1ab8c75478c6f73$export$3351871ee4b288b8 as PressResponder, $f1ab8c75478c6f73$export$cf75428e0b9ed1ea as ClearPressResponder};
//# sourceMappingURL=PressResponder.module.js.map
