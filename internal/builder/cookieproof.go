package builder

import (
	"fmt"
	"strings"
)

// RenderCookieProofSnippet returns the HTML snippet that the build pipeline
// injects into a site's <head> when CookieProof is enabled. Two parts:
//
//  1. The CookieProof loader script (https://consent.example.com/loader.js).
//     Multi-tenancy is keyed by data-domain. The loader synchronously fires the
//     GCM default-denied state on load.
//  2. A tiny relay that forwards consent:update + consent:gpc events from the
//     <cookie-consent> element to the site's own /t/consent endpoint, so the
//     visit_session can be lifted from anonymous to identified.
//
// siteID is the Atomicsite site ID (24-hex). siteDomain is the public hostname
// CookieProof keys configs by. trackPath is typically "/t" (configurable via
// cfg.TrackPath). All three are baked into the snippet at build time.
//
// Note: the relay POSTs same-origin so the atomicsite_fp cookie flows. Falls
// back to fetch+keepalive if sendBeacon refuses the body (rare; most browsers
// accept JSON via Beacon but it's been flaky historically).
func RenderCookieProofSnippet(siteID, siteDomain, trackPath string) string {
	if siteDomain == "" || siteID == "" {
		return ""
	}
	if trackPath == "" {
		trackPath = "/t"
	}
	trackPath = strings.TrimRight(trackPath, "/")

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		`<script async src="https://consent.example.com/loader.js" data-domain="%s"></script>`+"\n",
		escapeAttr(siteDomain),
	))
	b.WriteString("<script>\n")
	b.WriteString("(function(){\n")
	b.WriteString(fmt.Sprintf("  var TRACK = %q;\n", trackPath+"/consent"))
	b.WriteString(fmt.Sprintf("  var SITE_ID = %q;\n", siteID))
	b.WriteString("  function send(detail) {\n")
	b.WriteString("    try {\n")
	b.WriteString("      var body = JSON.stringify({\n")
	b.WriteString("        siteId: SITE_ID,\n")
	b.WriteString("        sessionId: localStorage.getItem('atomicsite_sid') || '',\n")
	b.WriteString("        consent: detail && detail.consent ? detail.consent : null,\n")
	b.WriteString("        page: location.href,\n")
	b.WriteString("        referrer: document.referrer\n")
	b.WriteString("      });\n")
	b.WriteString("      var sent = false;\n")
	b.WriteString("      if (navigator.sendBeacon) {\n")
	b.WriteString("        try { sent = navigator.sendBeacon(TRACK, new Blob([body], {type:'application/json'})); } catch(_){}\n")
	b.WriteString("      }\n")
	b.WriteString("      if (!sent) {\n")
	b.WriteString("        fetch(TRACK, {method:'POST', body: body, headers:{'Content-Type':'application/json'}, credentials:'include', keepalive: true});\n")
	b.WriteString("      }\n")
	b.WriteString("    } catch(e){}\n")
	b.WriteString("  }\n")
	b.WriteString("  document.addEventListener('consent:update', function(e){ send(e.detail); });\n")
	b.WriteString("  document.addEventListener('consent:gpc', function(e){ send(e.detail); });\n")
	b.WriteString("})();\n")
	b.WriteString("</script>\n")
	return b.String()
}
