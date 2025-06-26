define(function () {

  // Internal function that returns a bound version of the passed-in callback, to
  // be repeatedly applied in other Underscore functions.
  function bindCb(func, context) {
    if (context === void 0) return func;
    return function() {
      return func.apply(context, arguments);
    };
  }

  return bindCb;

});
