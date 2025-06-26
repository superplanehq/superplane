define(['exports', './_setup', './_getLength', './identity', './restArguments'], function (exports, _setup, _getLength, identity, restArguments) {

  // Internal function to work around a bug in IE < 9. See
  // https://github.com/jashkenas/underscore/issues/397.
  function removeGhostHead(array) {
    if (!_getLength(array)) delete array[0];
    return array;
  }

  // Internal function to wrap `Array.prototype` methods that modify
  // the context in place so they can be directly invoked as standalone
  // functions. Works for `pop`, `push`, `reverse`, `shift`, `sort`,
  // `splice` and `unshift`.
  function wrapArrayMutator(name, fixup) {
    var method = _setup.ArrayProto[name];
    fixup || (fixup = identity);
    return restArguments(function(array, args) {
      if (array != null) method.apply(array, args);
      return fixup(array);
    });
  }

  exports.default = wrapArrayMutator;
  exports.removeGhostHead = removeGhostHead;

  Object.defineProperty(exports, '__esModule', { value: true });

});
