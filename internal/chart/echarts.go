// Package chart generates offline ECharts HTML bandwidth charts.
package chart

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed echarts.min.js
var echartsJS []byte

type Series struct {
	Name   string
	Values []float64
	Peak   float64
}

type Config struct {
	Title   string
	XLabels []string
	Total   []float64
	Series  []Series
}

var bwUnits = []struct {
	d float64
	l string
}{{1e12, "Tbps"}, {1e9, "Gbps"}, {1e6, "Mbps"}, {1e3, "Kbps"}, {1, "bps"}}

func pickUnit(peak float64) (float64, string) {
	for _, u := range bwUnits {
		if peak >= u.d {
			return u.d, u.l
		}
	}
	return 1, "bps"
}

func WriteHTML(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	peak := 0.0
	for _, v := range cfg.Total {
		if v > peak {
			peak = v
		}
	}
	for _, s := range cfg.Series {
		if s.Peak > peak {
			peak = s.Peak
		}
	}
	divisor, unit := pickUnit(peak)

	series := make([]Series, len(cfg.Series))
	copy(series, cfg.Series)
	sort.Slice(series, func(i, j int) bool { return series[i].Peak > series[j].Peak })

	palette := []string{
		"#3b82f6", "#f97316", "#22c55e", "#a855f7", "#ef4444",
		"#06b6d4", "#eab308", "#ec4899", "#14b8a6", "#8b5cf6",
		"#f43f5e", "#84cc16", "#0ea5e9", "#fb923c", "#d946ef",
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	if len(cfg.Total) > 0 {
		fmt.Fprintf(&sb, `{name:"Total",type:"line",smooth:false,symbol:"none",lineStyle:{width:2},data:%s},`+"\n",
			floatJSON(cfg.Total, divisor))
	}
	for i, s := range series {
		color := palette[i%len(palette)]
		fmt.Fprintf(&sb, `{name:%s,type:"line",smooth:false,symbol:"none",lineStyle:{width:1.5,color:%s},itemStyle:{color:%s},areaStyle:{opacity:0.04},data:%s},`+"\n",
			jsonStr(s.Name), jsonStr(color), jsonStr(color), floatJSON(s.Values, divisor))
	}
	sb.WriteString("]")

	var legendNames []string
	if len(cfg.Total) > 0 {
		legendNames = append(legendNames, "Total")
	}
	for _, s := range series {
		legendNames = append(legendNames, s.Name)
	}

	longest := 0
	for _, n := range legendNames {
		if len(n) > longest {
			longest = len(n)
		}
	}
	legendWidth := longest*7 + 48
	if legendWidth < 120 {
		legendWidth = 120
	}
	if legendWidth > 280 {
		legendWidth = 280
	}
	gridRight := fmt.Sprintf(`"%dpx"`, legendWidth+16)

	title := cfg.Title
	if title == "" {
		title = "Bandwidth"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#111217;color:#d0d0d0;font-family:"Helvetica Neue",Arial,sans-serif}
body.light{background:#f5f6fa;color:#333}
#chart{width:100vw;height:100vh}
#toolbar{position:fixed;top:8px;right:12px;z-index:999;display:flex;gap:6px;align-items:center}
#toolbar button{padding:3px 10px;font-size:12px;border-radius:4px;cursor:pointer;
  border:1px solid #444;background:#1e2028;color:#ccc;white-space:nowrap}
body.light #toolbar button{background:#fff;color:#333;border-color:#ccc}
#toolbar button:hover{border-color:#3b82f6;color:#3b82f6}
#toolbar button.active{border-color:#f97316;color:#f97316;background:rgba(249,115,22,0.1)}
body.light #toolbar button.active{background:rgba(249,115,22,0.08)}
#hint{position:fixed;bottom:36px;left:12px;font-size:11px;color:#555;pointer-events:none}
body.light #hint{color:#999}
</style>
</head>
<body>
<div id="toolbar">
  <button id="btnMode">● Isolate</button>
  <button id="btnReset" style="display:none">Show All</button>
  <button id="btnTheme">☀ Light</button>
</div>
<div id="hint">Isolate: click legend to select/deselect (multi-select)</div>
<div id="chart"></div>
<script>%s</script>
<script>
(function(){
var T={
  dark:{bg:"#111217",ttBg:"rgba(22,24,30,0.95)",ttBorder:"#333",ttText:"#d0d0d0",
    axLine:"#2a2d35",axLabel:"#9ca3af",split:"#1f2128",totalColor:"#e0e0e0",
    zoomBg:"#1a1c23",zoomFill:"rgba(59,130,246,0.15)"},
  light:{bg:"#ffffff",ttBg:"rgba(255,255,255,0.97)",ttBorder:"#ddd",ttText:"#333",
    axLine:"#e5e7eb",axLabel:"#6b7280",split:"#f3f4f6",totalColor:"#374151",
    zoomBg:"#f3f4f6",zoomFill:"rgba(59,130,246,0.12)"}
};
var isDark=true,t=T.dark;
var allNames=%s;
var unit="%s";
var mode="isolate";
var selected={};
var hidden={};
var chart=echarts.init(document.getElementById("chart"),null,{renderer:"canvas"});
var rawSeries=%s;
var allSeries;
function buildAllSeries(){
  allSeries=rawSeries.map(function(s){
    if(s.name==="Total"){var ns=JSON.parse(JSON.stringify(s));ns.lineStyle=Object.assign({},ns.lineStyle,{color:t.totalColor});ns.itemStyle={color:t.totalColor};return ns;}
    return s;
  });
}
buildAllSeries();
function makeFormatter(filterSet){
  return function(params){
    var list=(filterSet&&Object.keys(filterSet).length)?params.filter(function(p){return filterSet[p.seriesName];}):params;
    if(!list.length)return"";
    var s=list[0].axisValue+"<br/>";
    list.forEach(function(p){if(p.value==null)return;s+='<span style="display:inline-block;width:10px;height:10px;border-radius:50%%;background:'+p.color+';margin-right:5px"></span>';s+=p.seriesName+": <b>"+p.value.toFixed(2)+" "+unit+"</b><br/>";});
    return s;
  };
}
function buildOption(seriesData,tooltipFilter){
  return {
    backgroundColor:t.bg,animation:false,
    title:{text:%s,left:"12px",top:"8px",textStyle:{color:t.axLabel,fontSize:14,fontWeight:"normal"}},
    tooltip:{trigger:"axis",axisPointer:{type:"line",lineStyle:{color:"rgba(128,128,128,0.2)"}},backgroundColor:t.ttBg,borderColor:t.ttBorder,borderWidth:1,textStyle:{color:t.ttText,fontSize:12},formatter:makeFormatter(tooltipFilter)},
    legend:{data:allNames,type:"scroll",orient:"vertical",right:8,top:40,bottom:40,textStyle:{color:t.axLabel,fontSize:11},icon:"roundRect",itemWidth:12,itemHeight:4,pageTextStyle:{color:t.axLabel}},
    grid:{left:"60px",right:%s,top:"40px",bottom:"50px",containLabel:false},
    xAxis:{type:"category",data:%s,boundaryGap:false,axisLine:{lineStyle:{color:t.axLine}},axisTick:{lineStyle:{color:t.axLine}},axisLabel:{color:t.axLabel,fontSize:11,formatter:function(v){return v.length>19?v.substring(0,19):v;}},splitLine:{show:false}},
    yAxis:{type:"value",axisLine:{show:false},axisTick:{show:false},axisLabel:{color:t.axLabel,fontSize:11,formatter:function(v){if(v===0)return"0";if(v>=1e12)return(v/1e12).toFixed(1)+"T";if(v>=1e9)return(v/1e9).toFixed(1)+"G";if(v>=1e6)return(v/1e6).toFixed(1)+"M";if(v>=1e3)return(v/1e3).toFixed(1)+"K";return v.toFixed(1);}},splitLine:{lineStyle:{color:t.split,type:"dashed"}},min:0},
    dataZoom:[{type:"inside",start:0,end:100},{type:"slider",start:0,end:100,height:18,bottom:"4px",borderColor:t.axLine,backgroundColor:t.zoomBg,fillerColor:t.zoomFill,handleStyle:{color:"#3b82f6"},textStyle:{color:t.axLabel,fontSize:10},right:%s}],
    series:seriesData
  };
}
function applyState(){
  var seriesData,tooltipFilter=null;
  if(mode==="isolate"){
    if(Object.keys(selected).length){tooltipFilter=selected;seriesData=allSeries.map(function(s){if(selected[s.name])return s;var ns=JSON.parse(JSON.stringify(s));ns.lineStyle=Object.assign({},ns.lineStyle,{opacity:0.07});if(ns.itemStyle)ns.itemStyle=Object.assign({},ns.itemStyle,{opacity:0.07});ns.areaStyle={opacity:0};return ns;});}
    else{seriesData=allSeries;}
  }else{
    seriesData=allSeries.map(function(s){if(!hidden[s.name])return s;var ns=JSON.parse(JSON.stringify(s));ns.lineStyle=Object.assign({},ns.lineStyle,{opacity:0});ns.itemStyle=Object.assign({},ns.itemStyle||{},{opacity:0});ns.areaStyle={opacity:0};return ns;});
  }
  chart.setOption(buildOption(seriesData,tooltipFilter),{replaceMerge:["series"]});
  document.getElementById("btnReset").style.display=((mode==="isolate"&&Object.keys(selected).length)||(mode==="hide"&&Object.keys(hidden).length))?"":"none";
}
var _suppress=false;
chart.on("legendselectchanged",function(params){
  if(_suppress)return;
  _suppress=true;chart.dispatchAction({type:"legendSelect",name:params.name});_suppress=false;
  var name=params.name;
  if(mode==="isolate"){if(selected[name])delete selected[name];else selected[name]=true;}
  else{if(hidden[name])delete hidden[name];else hidden[name]=true;}
  applyState();
});
document.getElementById("btnMode").addEventListener("click",function(){
  selected={};hidden={};mode=(mode==="isolate")?"hide":"isolate";
  this.textContent=mode==="isolate"?"● Isolate":"✕ Hide";
  this.classList.toggle("active",mode==="hide");
  document.getElementById("hint").textContent=mode==="isolate"?"Isolate: click legend to select/deselect (multi-select)":"Hide: click legend to hide/show lines";
  applyState();
});
document.getElementById("btnReset").addEventListener("click",function(){selected={};hidden={};applyState();});
document.getElementById("btnTheme").addEventListener("click",function(){
  isDark=!isDark;t=isDark?T.dark:T.light;document.body.classList.toggle("light",!isDark);
  this.textContent=isDark?"☀ Light":"🌙 Dark";buildAllSeries();applyState();
});
chart.setOption(buildOption(allSeries,null));
window.addEventListener("resize",function(){chart.resize();});
})();
</script>
</body>
</html>`,
		title, string(echartsJS),
		strSliceJSON(legendNames), unit, sb.String(),
		jsonStr(title), gridRight, strSliceJSON(cfg.XLabels), gridRight,
	)
	return os.WriteFile(path, []byte(html), 0o644)
}

func floatJSON(vals []float64, div float64) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range vals {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%.6g", v/div)
	}
	b.WriteByte(']')
	return b.String()
}

func strSliceJSON(vals []string) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range vals {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%q", v)
	}
	b.WriteByte(']')
	return b.String()
}

func jsonStr(s string) string { return fmt.Sprintf("%q", s) }
