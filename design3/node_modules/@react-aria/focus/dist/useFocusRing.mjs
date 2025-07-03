import {isFocusVisible as $isWE5$isFocusVisible, useFocusVisibleListener as $isWE5$useFocusVisibleListener, useFocus as $isWE5$useFocus, useFocusWithin as $isWE5$useFocusWithin} from "@react-aria/interactions";
import {useRef as $isWE5$useRef, useState as $isWE5$useState, useCallback as $isWE5$useCallback} from "react";



function $f7dceffc5ad7768b$export$4e328f61c538687f(props = {}) {
    let { autoFocus: autoFocus = false, isTextInput: isTextInput, within: within } = props;
    let state = (0, $isWE5$useRef)({
        isFocused: false,
        isFocusVisible: autoFocus || (0, $isWE5$isFocusVisible)()
    });
    let [isFocused, setFocused] = (0, $isWE5$useState)(false);
    let [isFocusVisibleState, setFocusVisible] = (0, $isWE5$useState)(()=>state.current.isFocused && state.current.isFocusVisible);
    let updateState = (0, $isWE5$useCallback)(()=>setFocusVisible(state.current.isFocused && state.current.isFocusVisible), []);
    let onFocusChange = (0, $isWE5$useCallback)((isFocused)=>{
        state.current.isFocused = isFocused;
        setFocused(isFocused);
        updateState();
    }, [
        updateState
    ]);
    (0, $isWE5$useFocusVisibleListener)((isFocusVisible)=>{
        state.current.isFocusVisible = isFocusVisible;
        updateState();
    }, [], {
        isTextInput: isTextInput
    });
    let { focusProps: focusProps } = (0, $isWE5$useFocus)({
        isDisabled: within,
        onFocusChange: onFocusChange
    });
    let { focusWithinProps: focusWithinProps } = (0, $isWE5$useFocusWithin)({
        isDisabled: !within,
        onFocusWithinChange: onFocusChange
    });
    return {
        isFocused: isFocused,
        isFocusVisible: isFocusVisibleState,
        focusProps: within ? focusWithinProps : focusProps
    };
}


export {$f7dceffc5ad7768b$export$4e328f61c538687f as useFocusRing};
//# sourceMappingURL=useFocusRing.module.js.map
