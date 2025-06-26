define(['./_setup', './_unmethodize'], function (_setup, _unmethodize) {

	var slice = _unmethodize(_setup.ArrayProto.slice);

	return slice;

});
