define(['./_tagTester', './_methodFingerprint', './_stringTagBug'], function (_tagTester, _methodFingerprint, _stringTagBug) {

	var isWeakMap = _stringTagBug.isIE11 ? _methodFingerprint.ie11fingerprint(_methodFingerprint.weakMapMethods) : _tagTester('WeakMap');

	return isWeakMap;

});
