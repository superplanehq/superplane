'use strict';

var react = require('react');

// This could've been more streamlined with internal state instead of abusing
// refs to such extent, but then composing hooks and components could not opt out of unnecessary renders.
function useResolvedElement(subscriber, refOrElement) {
  var lastReportRef = react.useRef(null);
  var refOrElementRef = react.useRef(null);
  refOrElementRef.current = refOrElement;
  var cbElementRef = react.useRef(null); // Calling re-evaluation after each render without using a dep array,
  // as the ref object's current value could've changed since the last render.

  react.useEffect(function () {
    evaluateSubscription();
  });
  var evaluateSubscription = react.useCallback(function () {
    var cbElement = cbElementRef.current;
    var refOrElement = refOrElementRef.current; // Ugly ternary. But smaller than an if-else block.

    var element = cbElement ? cbElement : refOrElement ? refOrElement instanceof Element ? refOrElement : refOrElement.current : null;

    if (lastReportRef.current && lastReportRef.current.element === element && lastReportRef.current.subscriber === subscriber) {
      return;
    }

    if (lastReportRef.current && lastReportRef.current.cleanup) {
      lastReportRef.current.cleanup();
    }

    lastReportRef.current = {
      element: element,
      subscriber: subscriber,
      // Only calling the subscriber, if there's an actual element to report.
      // Setting cleanup to undefined unless a subscriber returns one, as an existing cleanup function would've been just called.
      cleanup: element ? subscriber(element) : undefined
    };
  }, [subscriber]); // making sure we call the cleanup function on unmount

  react.useEffect(function () {
    return function () {
      if (lastReportRef.current && lastReportRef.current.cleanup) {
        lastReportRef.current.cleanup();
        lastReportRef.current = null;
      }
    };
  }, []);
  return react.useCallback(function (element) {
    cbElementRef.current = element;
    evaluateSubscription();
  }, [evaluateSubscription]);
}

// We're only using the first element of the size sequences, until future versions of the spec solidify on how
// exactly it'll be used for fragments in multi-column scenarios:
// From the spec:
// > The box size properties are exposed as FrozenArray in order to support elements that have multiple fragments,
// > which occur in multi-column scenarios. However the current definitions of content rect and border box do not
// > mention how those boxes are affected by multi-column layout. In this spec, there will only be a single
// > ResizeObserverSize returned in the FrozenArray, which will correspond to the dimensions of the first column.
// > A future version of this spec will extend the returned FrozenArray to contain the per-fragment size information.
// (https://drafts.csswg.org/resize-observer/#resize-observer-entry-interface)
//
// Also, testing these new box options revealed that in both Chrome and FF everything is returned in the callback,
// regardless of the "box" option.
// The spec states the following on this:
// > This does not have any impact on which box dimensions are returned to the defined callback when the event
// > is fired, it solely defines which box the author wishes to observe layout changes on.
// (https://drafts.csswg.org/resize-observer/#resize-observer-interface)
// I'm not exactly clear on what this means, especially when you consider a later section stating the following:
// > This section is non-normative. An author may desire to observe more than one CSS box.
// > In this case, author will need to use multiple ResizeObservers.
// (https://drafts.csswg.org/resize-observer/#resize-observer-interface)
// Which is clearly not how current browser implementations behave, and seems to contradict the previous quote.
// For this reason I decided to only return the requested size,
// even though it seems we have access to results for all box types.
// This also means that we get to keep the current api, being able to return a simple { width, height } pair,
// regardless of box option.
function extractSize(entry, boxProp, sizeType) {
  if (!entry[boxProp]) {
    if (boxProp === "contentBoxSize") {
      // The dimensions in `contentBoxSize` and `contentRect` are equivalent according to the spec.
      // See the 6th step in the description for the RO algorithm:
      // https://drafts.csswg.org/resize-observer/#create-and-populate-resizeobserverentry-h
      // > Set this.contentRect to logical this.contentBoxSize given target and observedBox of "content-box".
      // In real browser implementations of course these objects differ, but the width/height values should be equivalent.
      return entry.contentRect[sizeType === "inlineSize" ? "width" : "height"];
    }

    return undefined;
  } // A couple bytes smaller than calling Array.isArray() and just as effective here.


  return entry[boxProp][0] ? entry[boxProp][0][sizeType] : // TS complains about this, because the RO entry type follows the spec and does not reflect Firefox's current
  // behaviour of returning objects instead of arrays for `borderBoxSize` and `contentBoxSize`.
  // @ts-ignore
  entry[boxProp][sizeType];
}

function useResizeObserver(opts) {
  if (opts === void 0) {
    opts = {};
  }

  // Saving the callback as a ref. With this, I don't need to put onResize in the
  // effect dep array, and just passing in an anonymous function without memoising
  // will not reinstantiate the hook's ResizeObserver.
  var onResize = opts.onResize;
  var onResizeRef = react.useRef(undefined);
  onResizeRef.current = onResize;
  var round = opts.round || Math.round; // Using a single instance throughout the hook's lifetime

  var resizeObserverRef = react.useRef();

  var _useState = react.useState({
    width: undefined,
    height: undefined
  }),
      size = _useState[0],
      setSize = _useState[1]; // In certain edge cases the RO might want to report a size change just after
  // the component unmounted.


  var didUnmount = react.useRef(false);
  react.useEffect(function () {
    didUnmount.current = false;
    return function () {
      didUnmount.current = true;
    };
  }, []); // Using a ref to track the previous width / height to avoid unnecessary renders.

  var previous = react.useRef({
    width: undefined,
    height: undefined
  }); // This block is kinda like a useEffect, only it's called whenever a new
  // element could be resolved based on the ref option. It also has a cleanup
  // function.

  var refCallback = useResolvedElement(react.useCallback(function (element) {
    // We only use a single Resize Observer instance, and we're instantiating it on demand, only once there's something to observe.
    // This instance is also recreated when the `box` option changes, so that a new observation is fired if there was a previously observed element with a different box option.
    if (!resizeObserverRef.current || resizeObserverRef.current.box !== opts.box || resizeObserverRef.current.round !== round) {
      resizeObserverRef.current = {
        box: opts.box,
        round: round,
        instance: new ResizeObserver(function (entries) {
          var entry = entries[0];
          var boxProp = opts.box === "border-box" ? "borderBoxSize" : opts.box === "device-pixel-content-box" ? "devicePixelContentBoxSize" : "contentBoxSize";
          var reportedWidth = extractSize(entry, boxProp, "inlineSize");
          var reportedHeight = extractSize(entry, boxProp, "blockSize");
          var newWidth = reportedWidth ? round(reportedWidth) : undefined;
          var newHeight = reportedHeight ? round(reportedHeight) : undefined;

          if (previous.current.width !== newWidth || previous.current.height !== newHeight) {
            var newSize = {
              width: newWidth,
              height: newHeight
            };
            previous.current.width = newWidth;
            previous.current.height = newHeight;

            if (onResizeRef.current) {
              onResizeRef.current(newSize);
            } else {
              if (!didUnmount.current) {
                setSize(newSize);
              }
            }
          }
        })
      };
    }

    resizeObserverRef.current.instance.observe(element, {
      box: opts.box
    });
    return function () {
      if (resizeObserverRef.current) {
        resizeObserverRef.current.instance.unobserve(element);
      }
    };
  }, [opts.box, round]), opts.ref);
  return react.useMemo(function () {
    return {
      ref: refCallback,
      width: size.width,
      height: size.height
    };
  }, [refCallback, size.width, size.height]);
}

module.exports = useResizeObserver;
