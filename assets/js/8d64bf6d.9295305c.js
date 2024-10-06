"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[3183],{97486:(e,s,r)=>{r.r(s),r.d(s,{assets:()=>l,contentTitle:()=>a,default:()=>u,frontMatter:()=>n,metadata:()=>o,toc:()=>c});var i=r(74848),t=r(28453);const n={},a="Start",o={id:"sidebar/usage/cli/api/server/start",title:"Start",description:"To start the service:",source:"@site/docs/sidebar/usage/cli/api/server/start.md",sourceDirName:"sidebar/usage/cli/api/server",slug:"/sidebar/usage/cli/api/server/start",permalink:"/osapi/sidebar/usage/cli/api/server/start",draft:!1,unlisted:!1,tags:[],version:"current",frontMatter:{},sidebar:"testSidebar",previous:{title:"Server",permalink:"/osapi/sidebar/usage/cli/api/server/"},next:{title:"Client",permalink:"/osapi/sidebar/usage/cli/client/"}},l={},c=[{value:"Least Privilege Mode",id:"least-privilege-mode",level:2},{value:"ICMP Permissions",id:"icmp-permissions",level:3}];function d(e){const s={code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,t.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(s.header,{children:(0,i.jsx)(s.h1,{id:"start",children:"Start"})}),"\n",(0,i.jsx)(s.p,{children:"To start the service:"}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-bash",children:'$ osapi server start\n2:24AM INF server configuration debug=false server.port=8080 server.security.cors.allow_origins="[http://localhost:3001 https://retr0h.github.io]" database.driver_name=sqlite database.data_source_name="file:database.db?_journal=WAL&_timeout=5000&_fk=true" database.max_open_conns=1 database.max_idle_conns=1\n\u21e8 http server started on [::]:8080\n'})}),"\n",(0,i.jsx)(s.h2,{id:"least-privilege-mode",children:"Least Privilege Mode"}),"\n",(0,i.jsxs)(s.p,{children:["We aim to run this API service with the ",(0,i.jsx)(s.strong,{children:"least privilege mode"})," to maximize\nsecurity while reading data from Linux. This means that the API will only return\ndata that the running user has permission to access. The API is designed to\ngracefully skip over partitions or other system resources where permission\nerrors occur (e.g., due to lack of root access)."]}),"\n",(0,i.jsxs)(s.p,{children:["If your goal is to run the API with minimal limitations, you will need to run\nthe API daemon as ",(0,i.jsx)(s.code,{children:"root"}),"."]}),"\n",(0,i.jsx)(s.p,{children:"However, when run as a non-root user, the service will:"}),"\n",(0,i.jsxs)(s.ul,{children:["\n",(0,i.jsx)(s.li,{children:"Collect and return available disk usage statistics for partitions it has\npermission to access"}),"\n",(0,i.jsx)(s.li,{children:'Skip partitions or system paths that result in "permission denied" errors'}),"\n",(0,i.jsx)(s.li,{children:'Attempt to send an "unprivileged" ping via UDP'}),"\n"]}),"\n",(0,i.jsx)(s.p,{children:"Running as a regular user maintains a secure, restricted mode, but some\nfunctionality (such as access to certain system directories and files) will be\nlimited."}),"\n",(0,i.jsxs)(s.p,{children:["If full access to system resources is required (e.g., to access all disk\npartitions or perform privileged operations), running the API daemon as ",(0,i.jsx)(s.code,{children:"root"}),"\nis necessary."]}),"\n",(0,i.jsx)(s.h3,{id:"icmp-permissions",children:"ICMP Permissions"}),"\n",(0,i.jsx)(s.p,{children:"The API can still send pings without requiring root access, but on Linux, this\nrequires modifying system settings using the following sysctl command:"}),"\n",(0,i.jsx)(s.pre,{children:(0,i.jsx)(s.code,{className:"language-bash",children:'$ sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"\n'})}),"\n",(0,i.jsx)(s.p,{children:"Alternatively, running the API as root will allow full access to privileged\noperations like raw socket pinging."})]})}function u(e={}){const{wrapper:s}={...(0,t.R)(),...e.components};return s?(0,i.jsx)(s,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},28453:(e,s,r)=>{r.d(s,{R:()=>a,x:()=>o});var i=r(96540);const t={},n=i.createContext(t);function a(e){const s=i.useContext(n);return i.useMemo((function(){return"function"==typeof e?e(s):{...s,...e}}),[s,e])}function o(e){let s;return s=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:a(e.components),i.createElement(n.Provider,{value:s},e.children)}}}]);