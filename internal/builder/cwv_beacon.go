package builder

import (
	"fmt"
	"path/filepath"
)

// Sprint 5 quick-win (2026-05-24): Core Web Vitals measurement +
// reporting. Mirrors the engagement-beacon pattern: per-site JS
// asset written to public/, referenced via same-origin <script src>,
// CSP-safe under `script-src 'self'`.
//
// Implementation uses PerformanceObserver directly so we don't ship
// the full web-vitals npm package (the four metrics we care about
// are short to implement). Threshold ratings ('good' /
// 'needs-improvement' / 'poor') follow Google's published bands as
// of 2026 (LCP <2500ms / INP <200ms / CLS <0.1 / FCP <1800ms /
// TTFB <800ms = good; documented at web.dev/articles/vitals).

const CWVBeaconAssetName = "_atomic-cwv.js"

// CWVBeaconScript returns the vanilla JS body that observes the
// browser's Performance APIs, computes LCP / INP / CLS / FCP / TTFB,
// and posts each metric to /t/cwv via navigator.sendBeacon. siteID +
// trackPath are baked in at build time. consentGated mirrors the
// engagement-beacon behaviour: when true, the script waits for a
// `consent:init` event with analytics=true before reporting; when
// false (CookieProof not wired), reporting starts immediately
// (legitimate interest, same posture as the always-on nginx tail).
func CWVBeaconScript(siteID, trackPath string, consentGated bool) string {
	defaultAllowed := "true"
	if consentGated {
		defaultAllowed = "false"
	}
	return fmt.Sprintf(`(function(){
  var SITE_ID=%q;var TRACK_PATH=%q;var ALLOWED=%s;
  function device(){var ua=(navigator.userAgent||"").toLowerCase();if(/ipad|tablet|playbook|kindle/.test(ua))return "tablet";if(/mobile|android|iphone|phone/.test(ua))return "mobile";return "desktop";}
  function rate(m,v){
    if(m==="LCP"){return v<=2500?"good":(v<=4000?"needs-improvement":"poor");}
    if(m==="INP"){return v<=200?"good":(v<=500?"needs-improvement":"poor");}
    if(m==="CLS"){return v<=0.1?"good":(v<=0.25?"needs-improvement":"poor");}
    if(m==="FCP"){return v<=1800?"good":(v<=3000?"needs-improvement":"poor");}
    if(m==="TTFB"){return v<=800?"good":(v<=1800?"needs-improvement":"poor");}
    return "";
  }
  function send(metric,value){if(!ALLOWED||value==null||!isFinite(value))return;
    var p={siteId:SITE_ID,page:location.pathname,metric:metric,value:value,rating:rate(metric,value),device:device()};
    try{var b=new Blob([JSON.stringify(p)],{type:"application/json"});navigator.sendBeacon(TRACK_PATH+"/cwv",b);}catch(e){}
  }
  // TTFB: navigation entry's responseStart - startTime. Available
  // synchronously after the document parses; emit once.
  try{var nav=performance.getEntriesByType("navigation")[0];if(nav){send("TTFB",Math.max(0,nav.responseStart-nav.startTime));}}catch(e){}
  // FCP via PerformanceObserver paint-timing.
  try{var fcp=new PerformanceObserver(function(list){for(var i=0;i<list.getEntries().length;i++){var e=list.getEntries()[i];if(e.name==="first-contentful-paint"){send("FCP",e.startTime);fcp.disconnect();return;}}});fcp.observe({type:"paint",buffered:true});}catch(e){}
  // LCP: last entry of the largest-contentful-paint stream wins.
  // Report on visibilitychange/pagehide rather than on each new
  // candidate, so the value reflects the final LCP (web-vitals.js
  // does the same).
  var lcpValue=null;
  try{var lcp=new PerformanceObserver(function(list){var entries=list.getEntries();if(entries.length)lcpValue=entries[entries.length-1].renderTime||entries[entries.length-1].startTime;});lcp.observe({type:"largest-contentful-paint",buffered:true});}catch(e){}
  // CLS: cumulative layout shift score, summed across all observed
  // layout-shift entries that aren't preceded by recent user input.
  var clsValue=0;var clsSession=0;var clsSessionStart=0;var clsSessionEnd=0;
  try{var cls=new PerformanceObserver(function(list){var entries=list.getEntries();for(var i=0;i<entries.length;i++){var e=entries[i];if(e.hadRecentInput)continue;if(clsSession&&(e.startTime-clsSessionEnd>1000||e.startTime-clsSessionStart>5000)){if(clsSession>clsValue)clsValue=clsSession;clsSession=0;}if(!clsSession)clsSessionStart=e.startTime;clsSession+=e.value;clsSessionEnd=e.startTime;}if(clsSession>clsValue)clsValue=clsSession;});cls.observe({type:"layout-shift",buffered:true});}catch(e){}
  // INP: track interaction latencies; report the worst (98th percentile in
  // web-vitals.js spec; we approximate by max for small interaction counts
  // and decay to 98th when more than 50 interactions captured). For Slice
  // A we report the max which matches INP semantics on typical visits.
  var inpValue=0;
  try{var inp=new PerformanceObserver(function(list){var entries=list.getEntries();for(var i=0;i<entries.length;i++){var e=entries[i];var d=e.duration;if(typeof d==="number"&&d>inpValue)inpValue=d;}});inp.observe({type:"event",buffered:true,durationThreshold:40});}catch(e){}
  function flush(){send("LCP",lcpValue);send("CLS",clsValue);send("INP",inpValue);}
  document.addEventListener("consent:init",function(e){if(e&&e.detail&&e.detail.analytics)ALLOWED=true;});
  document.addEventListener("visibilitychange",function(){if(document.visibilityState==="hidden")flush();});
  window.addEventListener("pagehide",flush);
})();
`, siteID, trackPath, defaultAllowed)
}

// WriteCWVBeaconAsset writes the per-site CWV measurement JS to
// the workspace's public/ dir.
func WriteCWVBeaconAsset(workspaceDir, siteID, trackPath string, consentGated bool) error {
	publicDir := filepath.Join(workspaceDir, "public")
	if err := EnsureDir(publicDir); err != nil {
		return fmt.Errorf("ensure public dir: %w", err)
	}
	target := filepath.Join(publicDir, CWVBeaconAssetName)
	return WriteFile(target, CWVBeaconScript(siteID, trackPath, consentGated))
}

// RenderCWVBeacon returns the <script> tag that loads the per-site
// CWV asset. Same-origin defer load.
func RenderCWVBeacon() string {
	return fmt.Sprintf("  <script is:inline defer src=\"/%s\"></script>\n", CWVBeaconAssetName)
}
