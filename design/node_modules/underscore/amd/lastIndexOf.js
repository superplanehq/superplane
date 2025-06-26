define(['./_createIndexFinder', './findLastIndex'], function (_createIndexFinder, findLastIndex) {

	// Return the position of the last occurrence of an item in an array,
	// or -1 if the item is not included in the array.
	var lastIndexOf = _createIndexFinder(-1, findLastIndex);

	return lastIndexOf;

});
