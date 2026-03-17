'use strict';
//  app.js  assembly entry point 
// All module namespaces (PathProbe.*) are populated by the preceding <script>
// tags in index.html before DOMContentLoaded fires.

document.addEventListener('DOMContentLoaded', () => {
  // Leaflet default-icon path override (embedded assets, no CDN).
  if (typeof L !== 'undefined') {
    delete L.Icon.Default.prototype._getIconUrl;
    L.Icon.Default.mergeOptions({
      iconUrl:       '/images/marker-icon.png',
      iconRetinaUrl: '/images/marker-icon-2x.png',
      shadowUrl:     '/images/marker-shadow.png',
    });
  }

  // Initialise modules in dependency order.
  PathProbe.Form.init();
  PathProbe.ApiClient.fetchVersion();
  PathProbe.History.loadHistory();
  PathProbe.Theme.initTheme();    // apply saved theme (must run before locale)
  PathProbe.Locale.initLocale();  // apply saved locale (must run after DOM is ready)
});