package builder

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultAdminBaseURL is the default origin built sites' hydration script
// fetches /t/visitor from. Set by cmd/server/main.go on startup from
// cfg.BaseURL so we don't have to plumb config through every Build call.
// A site can override per-site via the analytics.admin_base_url setting.
var DefaultAdminBaseURL = ""

// RenderVisitorHydration emits a small inline <script> that fetches per-
// visitor metadata from /t/visitor (cross-origin to the admin host),
// exposes it on window.__SLAB_VISITOR__, walks every block tagged
// with [data-asp-when], evaluates the embedded DSL against the visitor
// fields, and removes the `hidden` attribute when the condition matches.
//
// Modelled byte-for-byte on RenderEngagementBeacon (layouts.go:223). Pure
// vanilla JS, no deps. Embeds the full DSL evaluator inline to avoid an
// extra round-trip; the grammar is mirrored 1:1 by Validate/Eval in the
// internal/personalization package, so anything the Go validator accepts
// the JS evaluator runs.
//
// Skip rendering this entirely when analytics.personalization_enabled is
// not "1": the fetch/eval cost (a single GET /t/visitor + DOM walk) is
// trivial, but keeping the script out of the page when the flag is off
// also keeps the eval engine's "tracker count" honest.
//
// adminBaseURL is the absolute origin of the admin host (e.g.
// "https://admin.example.com"). It is baked into the
// fetch URL because built sites live on different subdomains and cannot
// reach /t/visitor as a same-origin call. Empty falls back to relative
// "/t/visitor" for local-dev where built sites are on the same origin.
func VisitorHydrationScript(siteID, adminBaseURL, trackPath string) string {
	tp := trackPath
	if tp == "" {
		tp = "/t"
	}
	tp = strings.TrimRight(tp, "/")
	endpoint := fmt.Sprintf("%s%s/visitor", strings.TrimRight(adminBaseURL, "/"), tp)
	return fmt.Sprintf(`(function(){
  var SITE_ID=%q;var ENDPOINT=%q;
  function flat(v){var o={};if(!v||typeof v!=="object")return o;
    for(var k in v.anonymous||{})o[k]=v.anonymous[k];
    for(var k in v.identified||{})o[k]=v.identified[k];
    o.__returning=!!v.returning;o.__visitor_id=v.visitor_id||"";
    o.__identified=!!(v.identified&&Object.keys(v.identified).length);
    return o;}
  function evalCond(expr,vars){
    if(!expr)return true;
    var m=expr.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s+(present)\s*$/);
    if(m){return Object.prototype.hasOwnProperty.call(vars,m[1])&&vars[m[1]]!==""&&vars[m[1]]!=null;}
    m=expr.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s+(==|!=|>=|<=|>|<|in)\s+(.+?)\s*$/);
    if(!m)return false;
    var key=m[1],op=m[2],raw=m[3].trim();
    if(raw.length>=2&&raw[0]==='"'&&raw[raw.length-1]==='"')raw=raw.slice(1,-1);
    var lhs=vars[key];
    if(op==="in"){var list=raw.split(",").map(function(s){return s.trim();});return list.indexOf(String(lhs))>=0;}
    var ln=Number(lhs),rn=Number(raw);var numeric=!isNaN(ln)&&!isNaN(rn);
    switch(op){
      case "==":return numeric?ln===rn:String(lhs)===raw;
      case "!=":return numeric?ln!==rn:String(lhs)!==raw;
      case ">": return numeric&&ln>rn;
      case ">=":return numeric&&ln>=rn;
      case "<": return numeric&&ln<rn;
      case "<=":return numeric&&ln<=rn;
    }
    return false;
  }
  function apply(vars){
    var els=document.querySelectorAll("[data-asp-when]");
    for(var i=0;i<els.length;i++){var e=els[i];if(evalCond(e.getAttribute("data-asp-when"),vars))e.removeAttribute("hidden");}
  }
  fetch(ENDPOINT+"?site_id="+encodeURIComponent(SITE_ID),{credentials:"omit",mode:"cors",cache:"no-store"})
    .then(function(r){return r.ok?r.json():null;})
    .then(function(v){window.__SLAB_VISITOR__=v||{returning:false,anonymous:{},identified:{}};apply(flat(window.__SLAB_VISITOR__));})
    .catch(function(){window.__SLAB_VISITOR__={returning:false,anonymous:{},identified:{}};});
})();
`, siteID, endpoint)
}

// VisitorHydrationAssetName is the public/ filename. Stable per-site.
const VisitorHydrationAssetName = "_atomic-visitor.js"

// WriteVisitorHydrationAsset writes the per-site personalization-hydration
// JS file to the workspace's public/ dir so the page can reference it
// via a same-origin <script src>.
func WriteVisitorHydrationAsset(workspaceDir, siteID, adminBaseURL, trackPath string) error {
	publicDir := filepath.Join(workspaceDir, "public")
	if err := EnsureDir(publicDir); err != nil {
		return fmt.Errorf("ensure public dir: %w", err)
	}
	target := filepath.Join(publicDir, VisitorHydrationAssetName)
	return WriteFile(target, VisitorHydrationScript(siteID, adminBaseURL, trackPath))
}

func RenderVisitorHydration(siteID, adminBaseURL, trackPath string) string {
	return fmt.Sprintf(`  <script is:inline defer src="/%s"></script>
`, VisitorHydrationAssetName)
}
