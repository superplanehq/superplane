import {getInteractionModality as $507fabe10e71c6fb$export$630ff653c5ada6a9} from "./useFocusVisible.mjs";
import {getOwnerDocument as $k50bp$getOwnerDocument, getActiveElement as $k50bp$getActiveElement, runAfterTransition as $k50bp$runAfterTransition, focusWithoutScrolling as $k50bp$focusWithoutScrolling} from "@react-aria/utils";

/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an 'AS IS' BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 

function $3ad3f6e1647bc98d$export$80f3e147d781571c(element) {
    // If the user is interacting with a virtual cursor, e.g. screen reader, then
    // wait until after any animated transitions that are currently occurring on
    // the page before shifting focus. This avoids issues with VoiceOver on iOS
    // causing the page to scroll when moving focus if the element is transitioning
    // from off the screen.
    const ownerDocument = (0, $k50bp$getOwnerDocument)(element);
    const activeElement = (0, $k50bp$getActiveElement)(ownerDocument);
    if ((0, $507fabe10e71c6fb$export$630ff653c5ada6a9)() === 'virtual') {
        let lastFocusedElement = activeElement;
        (0, $k50bp$runAfterTransition)(()=>{
            // If focus did not move and the element is still in the document, focus it.
            if ((0, $k50bp$getActiveElement)(ownerDocument) === lastFocusedElement && element.isConnected) (0, $k50bp$focusWithoutScrolling)(element);
        });
    } else (0, $k50bp$focusWithoutScrolling)(element);
}


export {$3ad3f6e1647bc98d$export$80f3e147d781571c as focusSafely};
//# sourceMappingURL=focusSafely.module.js.map
