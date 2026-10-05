// Small enhancements; the site works without them.
(function () {
  "use strict";

  // Grow each preview frame to fit its content, as the platform's viewer does.
  function fit(frame) {
    try {
      var doc = frame.contentDocument;
      if (!doc || !doc.documentElement) return;
      var min = parseInt(frame.getAttribute("data-min"), 10) || 0;
      var h = Math.max(doc.documentElement.scrollHeight, doc.body ? doc.body.scrollHeight : 0);
      frame.style.height = Math.min(Math.max(h, min), 4000) + "px";
    } catch (e) {
      /* cross-origin (file://): keep the height from the marker */
    }
  }
  document.querySelectorAll("iframe.pv").forEach(function (f) {
    f.addEventListener("load", function () {
      fit(f);
      setTimeout(function () { fit(f); }, 400); // after fonts and async rendering settle
    });
  });

  // Filter the sidebar.
  var input = document.getElementById("filter");
  if (input) {
    input.addEventListener("input", function () {
      var q = input.value.trim().toLowerCase();
      document.querySelectorAll(".side .group").forEach(function (g) {
        var any = false;
        g.querySelectorAll("li").forEach(function (li) {
          var show = !q || li.textContent.toLowerCase().indexOf(q) !== -1;
          li.hidden = !show;
          any = any || show;
        });
        g.hidden = !any;
      });
    });
  }
})();
