define(function () {

  // A version of the `<=` operator that can be passed around as a function.
  function lessEqual(left, right) {
    return left <= right;
  }

  return lessEqual;

});
