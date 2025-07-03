"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
exports.getIsReactActEnvironment = getIsReactActEnvironment;
exports.setReactActEnvironment = setIsReactActEnvironment;

var testUtils = _interopRequireWildcard(require("react-dom/test-utils"));

function _getRequireWildcardCache(nodeInterop) { if (typeof WeakMap !== "function") return null; var cacheBabelInterop = new WeakMap(); var cacheNodeInterop = new WeakMap(); return (_getRequireWildcardCache = function (nodeInterop) { return nodeInterop ? cacheNodeInterop : cacheBabelInterop; })(nodeInterop); }

function _interopRequireWildcard(obj, nodeInterop) { if (!nodeInterop && obj && obj.__esModule) { return obj; } if (obj === null || typeof obj !== "object" && typeof obj !== "function") { return { default: obj }; } var cache = _getRequireWildcardCache(nodeInterop); if (cache && cache.has(obj)) { return cache.get(obj); } var newObj = {}; var hasPropertyDescriptor = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var key in obj) { if (key !== "default" && Object.prototype.hasOwnProperty.call(obj, key)) { var desc = hasPropertyDescriptor ? Object.getOwnPropertyDescriptor(obj, key) : null; if (desc && (desc.get || desc.set)) { Object.defineProperty(newObj, key, desc); } else { newObj[key] = obj[key]; } } } newObj.default = obj; if (cache) { cache.set(obj, newObj); } return newObj; }

const domAct = testUtils.act;

function getGlobalThis() {
  /* istanbul ignore else */
  if (typeof globalThis !== 'undefined') {
    return globalThis;
  }
  /* istanbul ignore next */


  if (typeof self !== 'undefined') {
    return self;
  }
  /* istanbul ignore next */


  if (typeof window !== 'undefined') {
    return window;
  }
  /* istanbul ignore next */


  if (typeof global !== 'undefined') {
    return global;
  }
  /* istanbul ignore next */


  throw new Error('unable to locate global object');
}

function setIsReactActEnvironment(isReactActEnvironment) {
  getGlobalThis().IS_REACT_ACT_ENVIRONMENT = isReactActEnvironment;
}

function getIsReactActEnvironment() {
  return getGlobalThis().IS_REACT_ACT_ENVIRONMENT;
}

function withGlobalActEnvironment(actImplementation) {
  return callback => {
    const previousActEnvironment = getIsReactActEnvironment();
    setIsReactActEnvironment(true);

    try {
      // The return value of `act` is always a thenable.
      let callbackNeedsToBeAwaited = false;
      const actResult = actImplementation(() => {
        const result = callback();

        if (result !== null && typeof result === 'object' && typeof result.then === 'function') {
          callbackNeedsToBeAwaited = true;
        }

        return result;
      });

      if (callbackNeedsToBeAwaited) {
        const thenable = actResult;
        return {
          then: (resolve, reject) => {
            thenable.then(returnValue => {
              setIsReactActEnvironment(previousActEnvironment);
              resolve(returnValue);
            }, error => {
              setIsReactActEnvironment(previousActEnvironment);
              reject(error);
            });
          }
        };
      } else {
        setIsReactActEnvironment(previousActEnvironment);
        return actResult;
      }
    } catch (error) {
      // Can't be a `finally {}` block since we don't know if we have to immediately restore IS_REACT_ACT_ENVIRONMENT
      // or if we have to await the callback first.
      setIsReactActEnvironment(previousActEnvironment);
      throw error;
    }
  };
}

const act = withGlobalActEnvironment(domAct);
var _default = act;
/* eslint no-console:0 */

exports.default = _default;