// ==UserScript==
// @name         OpenRouter Scroll Fix
// @match        https://openrouter.ai/*
// @run-at       document-start
// @grant        none
// ==/UserScript==

(function() {
  var descriptor = Object.getOwnPropertyDescriptor(Element.prototype, 'scrollTop');
  Object.defineProperty(Element.prototype, 'scrollTop', {
    get: descriptor.get,
    set: function(value) {
      var max = this.scrollHeight - this.clientHeight;
      var current = descriptor.get.call(this) || 0;
      if (max - value < 1 && value - current > 50) return;
      descriptor.set.call(this, value);
    },
    configurable: true
  });
})();
