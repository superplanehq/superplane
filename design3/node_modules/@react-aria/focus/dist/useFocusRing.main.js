var $7eRoM$reactariainteractions = require("@react-aria/interactions");
var $7eRoM$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useFocusRing", () => $581a96d6eb128c1b$export$4e328f61c538687f);


function $581a96d6eb128c1b$export$4e328f61c538687f(props = {}) {
    let { autoFocus: autoFocus = false, isTextInput: isTextInput, within: within } = props;
    let state = (0, $7eRoM$react.useRef)({
        isFocused: false,
        isFocusVisible: autoFocus || (0, $7eRoM$reactariainteractions.isFocusVisible)()
    });
    let [isFocused, setFocused] = (0, $7eRoM$react.useState)(false);
    let [isFocusVisibleState, setFocusVisible] = (0, $7eRoM$react.useState)(()=>state.current.isFocused && state.current.isFocusVisible);
    let updateState = (0, $7eRoM$react.useCallback)(()=>setFocusVisible(state.current.isFocused && state.current.isFocusVisible), []);
    let onFocusChange = (0, $7eRoM$react.useCallback)((isFocused)=>{
        state.current.isFocused = isFocused;
        setFocused(isFocused);
        updateState();
    }, [
        updateState
    ]);
    (0, $7eRoM$reactariainteractions.useFocusVisibleListener)((isFocusVisible)=>{
        state.current.isFocusVisible = isFocusVisible;
        updateState();
    }, [], {
        isTextInput: isTextInput
    });
    let { focusProps: focusProps } = (0, $7eRoM$reactariainteractions.useFocus)({
        isDisabled: within,
        onFocusChange: onFocusChange
    });
    let { focusWithinProps: focusWithinProps } = (0, $7eRoM$reactariainteractions.useFocusWithin)({
        isDisabled: !within,
        onFocusWithinChange: onFocusChange
    });
    return {
        isFocused: isFocused,
        isFocusVisible: isFocusVisibleState,
        focusProps: within ? focusWithinProps : focusProps
    };
}


//# sourceMappingURL=useFocusRing.main.js.map
