/* @ds-bundle: {"format":4,"namespace":"Mini","components":[{"name":"Button"},{"name":"Badge"}]} */
(function () {
  if (!window.Stub) throw new Error("library not loaded before the bundle");
  function el(tag, cls, text) { var e = document.createElement(tag); e.className = cls; if (text) e.textContent = text; return e; }
  window.Mini = {
    Button: function (label) {
      var b = el("button", "mini-btn", label);
      b.style.backgroundImage = "url(/_blob/0123456789abcdef0123456789abcdef)";
      return b;
    },
    Badge: function (label) { return el("span", "mini-card", label); }
  };
})();
