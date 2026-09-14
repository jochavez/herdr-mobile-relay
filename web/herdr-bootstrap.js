const e = new URL(window.__HERDR_ENTRY__ || "/builds/0.22.0-383-79be5d7b605cd55a/index.html", location);
  e.search = location.search;
  e.hash = location.hash;
  location.replace(e);
