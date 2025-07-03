import {createSyntheticEvent as $8a9cb279dc87e130$export$525bc4921d56d4a, setEventTarget as $8a9cb279dc87e130$export$c2b7abe5d61ec696, useSyntheticBlurEvent as $8a9cb279dc87e130$export$715c682d09d639cc} from "./utils.module.js";
import {useRef as $3b9Q0$useRef, useCallback as $3b9Q0$useCallback} from "react";
import {useGlobalListeners as $3b9Q0$useGlobalListeners, getOwnerDocument as $3b9Q0$getOwnerDocument, getActiveElement as $3b9Q0$getActiveElement, getEventTarget as $3b9Q0$getEventTarget, nodeContains as $3b9Q0$nodeContains} from "@react-aria/utils";

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
 */ // Portions of the code in this file are based on code from react.
// Original licensing for the following can be found in the
// NOTICE file in the root directory of this source tree.
// See https://github.com/facebook/react/tree/cc7c1aece46a6b69b41958d731e0fd27c94bfc6c/packages/react-interactions



function $9ab94262bd0047c7$export$420e68273165f4ec(props) {
    let { isDisabled: isDisabled, onBlurWithin: onBlurWithin, onFocusWithin: onFocusWithin, onFocusWithinChange: onFocusWithinChange } = props;
    let state = (0, $3b9Q0$useRef)({
        isFocusWithin: false
    });
    let { addGlobalListener: addGlobalListener, removeAllGlobalListeners: removeAllGlobalListeners } = (0, $3b9Q0$useGlobalListeners)();
    let onBlur = (0, $3b9Q0$useCallback)((e)=>{
        // Ignore events bubbling through portals.
        if (!e.currentTarget.contains(e.target)) return;
        // We don't want to trigger onBlurWithin and then immediately onFocusWithin again
        // when moving focus inside the element. Only trigger if the currentTarget doesn't
        // include the relatedTarget (where focus is moving).
        if (state.current.isFocusWithin && !e.currentTarget.contains(e.relatedTarget)) {
            state.current.isFocusWithin = false;
            removeAllGlobalListeners();
            if (onBlurWithin) onBlurWithin(e);
            if (onFocusWithinChange) onFocusWithinChange(false);
        }
    }, [
        onBlurWithin,
        onFocusWithinChange,
        state,
        removeAllGlobalListeners
    ]);
    let onSyntheticFocus = (0, $8a9cb279dc87e130$export$715c682d09d639cc)(onBlur);
    let onFocus = (0, $3b9Q0$useCallback)((e)=>{
        // Ignore events bubbling through portals.
        if (!e.currentTarget.contains(e.target)) return;
        // Double check that document.activeElement actually matches e.target in case a previously chained
        // focus handler already moved focus somewhere else.
        const ownerDocument = (0, $3b9Q0$getOwnerDocument)(e.target);
        const activeElement = (0, $3b9Q0$getActiveElement)(ownerDocument);
        if (!state.current.isFocusWithin && activeElement === (0, $3b9Q0$getEventTarget)(e.nativeEvent)) {
            if (onFocusWithin) onFocusWithin(e);
            if (onFocusWithinChange) onFocusWithinChange(true);
            state.current.isFocusWithin = true;
            onSyntheticFocus(e);
            // Browsers don't fire blur events when elements are removed from the DOM.
            // However, if a focus event occurs outside the element we're tracking, we
            // can manually fire onBlur.
            let currentTarget = e.currentTarget;
            addGlobalListener(ownerDocument, 'focus', (e)=>{
                if (state.current.isFocusWithin && !(0, $3b9Q0$nodeContains)(currentTarget, e.target)) {
                    let nativeEvent = new ownerDocument.defaultView.FocusEvent('blur', {
                        relatedTarget: e.target
                    });
                    (0, $8a9cb279dc87e130$export$c2b7abe5d61ec696)(nativeEvent, currentTarget);
                    let event = (0, $8a9cb279dc87e130$export$525bc4921d56d4a)(nativeEvent);
                    onBlur(event);
                }
            }, {
                capture: true
            });
        }
    }, [
        onFocusWithin,
        onFocusWithinChange,
        onSyntheticFocus,
        addGlobalListener,
        onBlur
    ]);
    if (isDisabled) return {
        focusWithinProps: {
            // These cannot be null, that would conflict in mergeProps
            onFocus: undefined,
            onBlur: undefined
        }
    };
    return {
        focusWithinProps: {
            onFocus: onFocus,
            onBlur: onBlur
        }
    };
}


export {$9ab94262bd0047c7$export$420e68273165f4ec as useFocusWithin};
//# sourceMappingURL=useFocusWithin.module.js.map
