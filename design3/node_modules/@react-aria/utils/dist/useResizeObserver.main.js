var $aM4zL$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useResizeObserver", () => $37733e1652f47193$export$683480f191c0e3ea);

function $37733e1652f47193$var$hasResizeObserver() {
    return typeof window.ResizeObserver !== 'undefined';
}
function $37733e1652f47193$export$683480f191c0e3ea(options) {
    const { ref: ref, box: box, onResize: onResize } = options;
    (0, $aM4zL$react.useEffect)(()=>{
        let element = ref === null || ref === void 0 ? void 0 : ref.current;
        if (!element) return;
        if (!$37733e1652f47193$var$hasResizeObserver()) {
            window.addEventListener('resize', onResize, false);
            return ()=>{
                window.removeEventListener('resize', onResize, false);
            };
        } else {
            const resizeObserverInstance = new window.ResizeObserver((entries)=>{
                if (!entries.length) return;
                onResize();
            });
            resizeObserverInstance.observe(element, {
                box: box
            });
            return ()=>{
                if (element) resizeObserverInstance.unobserve(element);
            };
        }
    }, [
        onResize,
        ref,
        box
    ]);
}


//# sourceMappingURL=useResizeObserver.main.js.map
