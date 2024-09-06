"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[474],{51210:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>u,frontMatter:()=>r,metadata:()=>o,toc:()=>l});var s=n(74848),i=n(28453);const r={},a="System",o={id:"sidebar/usage/cli/client/system",title:"System",description:"Status",source:"@site/docs/sidebar/usage/cli/client/system.md",sourceDirName:"sidebar/usage/cli/client",slug:"/sidebar/usage/cli/client/system",permalink:"/osapi/sidebar/usage/cli/client/system",draft:!1,unlisted:!1,tags:[],version:"current",frontMatter:{},sidebar:"testSidebar",previous:{title:"Ping",permalink:"/osapi/sidebar/usage/cli/client/ping"},next:{title:"Server",permalink:"/osapi/sidebar/usage/cli/server"}},c={},l=[{value:"Status",id:"status",level:2}];function d(e){const t={admonition:"admonition",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"system",children:"System"})}),"\n",(0,s.jsx)(t.h2,{id:"status",children:"Status"}),"\n",(0,s.jsx)(t.admonition,{type:"note",children:(0,s.jsx)(t.p,{children:"The following disk layout took place on a Docker container."})}),"\n",(0,s.jsx)(t.p,{children:"Get the system status:"}),"\n",(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{className:"language-bash",children:"$ osapi client system status\n9:30PM INF client configuration debug=false client.url=http://0.0.0.0:8080\n\n  Hostname: ca946721bd50\n  Uptime: 3 days, 4 hours, 41 minutes\n  Load Average (1m, 5m, 15m): 0.00, 0.00, 0.07\n  Memory: 13 GB used / 15 GB total / 0 GB free\n\n  Disks:\n\n  \u250f\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2533\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2533\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2533\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2513\n  \u2503     DISK NAME      \u2503       TOTAL        \u2503        USED        \u2503        FREE        \u2503\n  \u2523\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u254b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u254b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u254b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u252b\n  \u2503 /etc/resolv.conf   \u2503 251 GB             \u2503 19 GB              \u2503 219 GB             \u2503\n  \u2503 /etc/hostname      \u2503 251 GB             \u2503 19 GB              \u2503 219 GB             \u2503\n  \u2503 /etc/hosts         \u2503 251 GB             \u2503 19 GB              \u2503 219 GB             \u2503\n  \u2517\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u253b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u253b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u253b\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u251b\n"})})]})}function u(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(d,{...e})}):d(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>a,x:()=>o});var s=n(96540);const i={},r=s.createContext(i);function a(e){const t=s.useContext(r);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:a(e.components),s.createElement(r.Provider,{value:t},e.children)}}}]);