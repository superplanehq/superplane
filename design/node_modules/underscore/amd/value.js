define(function () {

  // Extract the result from a wrapped (and possibly chained) object.
  // This function is also aliased as `valueOf` and `toJSON`, which provide
  // unwrapping proxies for some methods used in engine operations such as
  // arithmetic and JSON stringification.
  function value(wrapper) {
    return wrapper._wrapped || wrapper;
  }

  return value;

});
