"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[371],{21006:(e,s,a)=>{a.r(s),a.d(s,{assets:()=>N,contentTitle:()=>f,default:()=>k,frontMatter:()=>v,metadata:()=>b,toc:()=>_});var i=a(74848),t=a(28453),r=a(91366),n=a.n(r),l=(a(6050),a(57742)),m=a.n(l),o=(a(67792),a(27362)),c=a.n(o),d=(a(36683),a(81124)),p=a.n(d),h=a(60674),u=a.n(h),x=a(23397),y=a.n(x),g=a(51107),j=(a(77675),a(19365));const v={id:"retrieve-system-status",title:"Retrieve system status",description:"Get the current status of the system including hostname, uptime, load averages, memory, and disk usage.",sidebar_label:"Retrieve system status",hide_title:!0,hide_table_of_contents:!0,api:"eJy9Vm1v0zAQ/iuWvwBSaJNuhbJvk3gRiDdt49OYKte5NmZJHGxnrKr637mz0zZJx7oJxKap6XJ+7rl77sUrnoKVRlVO6ZKf8HfgmMuAydoYKB2zTrjaMj33/7VL66BgqpR5napywTJtXSkKiFiNCPSZa5EycQNGLMBGrIBCm2XERJmyVNlrVlt8MeARd2Jh+cklP/eY03PvaHr69f00eJnqCkGIluVXETdgK3wEPLPiozimjy71U/bh/Mtnpmc/QDomdemEKonkjvkTuwlIlXNtCg9PZMgawyVQUVW5kv7N8Icl5BW3MoNC0JNbVoC+ghM8WBmi6VTgtUlHy9I6gxzQskv2AjltrLvZJTpwK4oqp+PF8nmuyvr2uQWDSeXriIdMP8xFsL3PQcxSsUShjpFObfAhYQU6dECuSMxpI+Zd0e87bCqkUwUMU82SiI1DGSTjxoMd7CUwwTctR2VdzDDovqOPLXQPTsHlwrot+U6M8eBohMGM/xK7Q7sFPpogePK36Mkf4cfrNZX/z1oZSKljfJKaeDaesUOccl5R8nHaaIYnQwceVu+Ttwv92W+PrkhOO5G3ABX2zuKOYC/IrpkAiMhmy350k6PJ5EVM+ZsbgAdAvkWzexFH8auXyZjkri1l6yDiNzS7F/E4eXV8FB/3RQhJaIg33loihGxS/mnqHc7+6+1s/Je59xPXVkLC3bGN493PI1N2CHo07kA/Rt9HQf9BFh9K4/aqb7Qd09th2ht1265p5GsJG9ZV2FZ8vSbo8V3b6I0x2N4GcDbDDW2hZjKG/fPvVg6Qn8PL4JS1vm/2gT+LT8IxLf3GT7vL4a1QOWrt9CYQaF8DQigDX+SAyza3D+CRpooeqTjDGSZmunY7PhFKD1LNl5S0X5mSGe7yAnc/XUfmntBBkhuBPTep04eU3sU2IXSg3yb9Egppb8Bb5eFlP2vuKr4+/Ah2mcZjfAFeQOEy/DIMaRxuKyKseLoUrXhtsMt55lx1Mhwmo5eDGH+Tk0k8ifcnt5bXDC9O7Gmupcif8TUSoimyX5VfzsmO/lIt6wJTGubMd9ok5DyYJeSO76LaHWuuIREP1xxekcgYYoUZL4Sv3ObN2UaMfuF3GK12ffA/756hFBzcumGV4zXR36uMH6xBnUse/JEsgTjmlDzSq9VqJix8M/l6Tf/+WQMt2Et8vBFGiRnl7PIKITMQKWaLBL2GJZW/lFBRDdyIvCYKe61P2m3r5d2bC6yh3/CK9qU=",sidebar_class_name:"get api-method",info_path:"gen/api/osapi-api-server",custom_edit_url:null},f=void 0,b={id:"gen/api/retrieve-system-status",title:"Retrieve system status",description:"Get the current status of the system including hostname, uptime, load averages, memory, and disk usage.",source:"@site/docs/gen/api/retrieve-system-status.api.mdx",sourceDirName:"gen/api",slug:"/gen/api/retrieve-system-status",permalink:"/osapi/gen/api/retrieve-system-status",draft:!1,unlisted:!1,editUrl:null,tags:[],version:"current",frontMatter:{id:"retrieve-system-status",title:"Retrieve system status",description:"Get the current status of the system including hostname, uptime, load averages, memory, and disk usage.",sidebar_label:"Retrieve system status",hide_title:!0,hide_table_of_contents:!0,api:"eJy9Vm1v0zAQ/iuWvwBSaJNuhbJvk3gRiDdt49OYKte5NmZJHGxnrKr637mz0zZJx7oJxKap6XJ+7rl77sUrnoKVRlVO6ZKf8HfgmMuAydoYKB2zTrjaMj33/7VL66BgqpR5napywTJtXSkKiFiNCPSZa5EycQNGLMBGrIBCm2XERJmyVNlrVlt8MeARd2Jh+cklP/eY03PvaHr69f00eJnqCkGIluVXETdgK3wEPLPiozimjy71U/bh/Mtnpmc/QDomdemEKonkjvkTuwlIlXNtCg9PZMgawyVQUVW5kv7N8Icl5BW3MoNC0JNbVoC+ghM8WBmi6VTgtUlHy9I6gxzQskv2AjltrLvZJTpwK4oqp+PF8nmuyvr2uQWDSeXriIdMP8xFsL3PQcxSsUShjpFObfAhYQU6dECuSMxpI+Zd0e87bCqkUwUMU82SiI1DGSTjxoMd7CUwwTctR2VdzDDovqOPLXQPTsHlwrot+U6M8eBohMGM/xK7Q7sFPpogePK36Mkf4cfrNZX/z1oZSKljfJKaeDaesUOccl5R8nHaaIYnQwceVu+Ttwv92W+PrkhOO5G3ABX2zuKOYC/IrpkAiMhmy350k6PJ5EVM+ZsbgAdAvkWzexFH8auXyZjkri1l6yDiNzS7F/E4eXV8FB/3RQhJaIg33loihGxS/mnqHc7+6+1s/Je59xPXVkLC3bGN493PI1N2CHo07kA/Rt9HQf9BFh9K4/aqb7Qd09th2ht1265p5GsJG9ZV2FZ8vSbo8V3b6I0x2N4GcDbDDW2hZjKG/fPvVg6Qn8PL4JS1vm/2gT+LT8IxLf3GT7vL4a1QOWrt9CYQaF8DQigDX+SAyza3D+CRpooeqTjDGSZmunY7PhFKD1LNl5S0X5mSGe7yAnc/XUfmntBBkhuBPTep04eU3sU2IXSg3yb9Egppb8Bb5eFlP2vuKr4+/Ah2mcZjfAFeQOEy/DIMaRxuKyKseLoUrXhtsMt55lx1Mhwmo5eDGH+Tk0k8ifcnt5bXDC9O7Gmupcif8TUSoimyX5VfzsmO/lIt6wJTGubMd9ok5DyYJeSO76LaHWuuIREP1xxekcgYYoUZL4Sv3ObN2UaMfuF3GK12ffA/756hFBzcumGV4zXR36uMH6xBnUse/JEsgTjmlDzSq9VqJix8M/l6Tf/+WQMt2Et8vBFGiRnl7PIKITMQKWaLBL2GJZW/lFBRDdyIvCYKe61P2m3r5d2bC6yh3/CK9qU=",sidebar_class_name:"get api-method",info_path:"gen/api/osapi-api-server",custom_edit_url:null},sidebar:"testSidebar",previous:{title:"System",permalink:"/osapi/gen/api/system-status-api-system-operations"}},N={},_=[];function M(e){const s={p:"p",...(0,t.R)(),...e.components},{Details:a}=s;return a||function(e,s){throw new Error("Expected "+(s?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}("Details",!0),(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(g.default,{as:"h1",className:"openapi__heading",children:"Retrieve system status"}),"\n",(0,i.jsx)(m(),{method:"get",path:"/system/status"}),"\n",(0,i.jsx)(s.p,{children:"Get the current status of the system including hostname, uptime, load averages, memory, and disk usage."}),"\n",(0,i.jsx)("div",{children:(0,i.jsx)("div",{children:(0,i.jsxs)(n(),{label:void 0,id:void 0,children:[(0,i.jsxs)(j.default,{label:"200",value:"200",children:[(0,i.jsx)("div",{children:(0,i.jsx)(s.p,{children:"A JSON object containing the system's status information."})}),(0,i.jsx)("div",{children:(0,i.jsx)(c(),{className:"openapi-tabs__mime",schemaType:"response",children:(0,i.jsx)(j.default,{label:"application/json",value:"application/json",children:(0,i.jsxs)(y(),{className:"openapi-tabs__schema",children:[(0,i.jsx)(j.default,{label:"Schema",value:"Schema",children:(0,i.jsxs)(a,{style:{},className:"openapi-markdown__details response","data-collapsed":!1,open:!0,children:[(0,i.jsx)("summary",{style:{},className:"openapi-markdown__details-summary-response",children:(0,i.jsx)("strong",{children:(0,i.jsx)(s.p,{children:"Schema"})})}),(0,i.jsx)("div",{style:{textAlign:"left",marginLeft:"1rem"}}),(0,i.jsxs)("ul",{style:{marginLeft:"1rem"},children:[(0,i.jsx)(u(),{collapsible:!1,name:"hostname",required:!0,schemaName:"string",qualifierMessage:void 0,schema:{type:"string",description:"The hostname of the system.",example:"my-linux-server"}}),(0,i.jsx)(u(),{collapsible:!1,name:"uptime",required:!0,schemaName:"string",qualifierMessage:void 0,schema:{type:"string",description:"The uptime of the system.",example:"0 days, 4 hours, 1 minute"}}),(0,i.jsx)(u(),{collapsible:!0,className:"schemaItem",children:(0,i.jsxs)(a,{style:{},className:"openapi-markdown__details",children:[(0,i.jsx)("summary",{style:{},children:(0,i.jsxs)("span",{className:"openapi-schema__container",children:[(0,i.jsx)("strong",{className:"openapi-schema__property",children:(0,i.jsx)(s.p,{children:"load_average"})}),(0,i.jsx)("span",{className:"openapi-schema__name",children:(0,i.jsx)(s.p,{children:"object"})}),(0,i.jsx)("span",{className:"openapi-schema__divider"}),(0,i.jsx)("span",{className:"openapi-schema__required",children:(0,i.jsx)(s.p,{children:"required"})})]})}),(0,i.jsxs)("div",{style:{marginLeft:"1rem"},children:[(0,i.jsx)("div",{style:{marginTop:".5rem",marginBottom:".5rem"},children:(0,i.jsx)(s.p,{children:"The system load averages for 1, 5, and 15 minutes."})}),(0,i.jsx)(u(),{collapsible:!1,name:"1min",required:!0,schemaName:"number",qualifierMessage:void 0,schema:{type:"number",description:"Load average for the last 1 minute.",example:.32}}),(0,i.jsx)(u(),{collapsible:!1,name:"5min",required:!0,schemaName:"number",qualifierMessage:void 0,schema:{type:"number",description:"Load average for the last 5 minutes.",example:.28}}),(0,i.jsx)(u(),{collapsible:!1,name:"15min",required:!0,schemaName:"number",qualifierMessage:void 0,schema:{type:"number",description:"Load average for the last 15 minutes.",example:.25}})]})]})}),(0,i.jsx)(u(),{collapsible:!0,className:"schemaItem",children:(0,i.jsxs)(a,{style:{},className:"openapi-markdown__details",children:[(0,i.jsx)("summary",{style:{},children:(0,i.jsxs)("span",{className:"openapi-schema__container",children:[(0,i.jsx)("strong",{className:"openapi-schema__property",children:(0,i.jsx)(s.p,{children:"memory"})}),(0,i.jsx)("span",{className:"openapi-schema__name",children:(0,i.jsx)(s.p,{children:"object"})}),(0,i.jsx)("span",{className:"openapi-schema__divider"}),(0,i.jsx)("span",{className:"openapi-schema__required",children:(0,i.jsx)(s.p,{children:"required"})})]})}),(0,i.jsxs)("div",{style:{marginLeft:"1rem"},children:[(0,i.jsx)("div",{style:{marginTop:".5rem",marginBottom:".5rem"},children:(0,i.jsx)(s.p,{children:"Memory usage information."})}),(0,i.jsx)(u(),{collapsible:!1,name:"total",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Total memory in bytes.",example:8388608}}),(0,i.jsx)(u(),{collapsible:!1,name:"free",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Free memory in bytes.",example:2097152}}),(0,i.jsx)(u(),{collapsible:!1,name:"used",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Used memory in bytes.",example:4194304}})]})]})}),(0,i.jsx)(u(),{collapsible:!0,className:"schemaItem",children:(0,i.jsxs)(a,{style:{},className:"openapi-markdown__details",children:[(0,i.jsx)("summary",{style:{},children:(0,i.jsxs)("span",{className:"openapi-schema__container",children:[(0,i.jsx)("strong",{className:"openapi-schema__property",children:(0,i.jsx)(s.p,{children:"disk"})}),(0,i.jsx)("span",{className:"openapi-schema__name",children:(0,i.jsx)(s.p,{children:"object"})}),(0,i.jsx)("span",{className:"openapi-schema__divider"}),(0,i.jsx)("span",{className:"openapi-schema__required",children:(0,i.jsx)(s.p,{children:"required"})})]})}),(0,i.jsxs)("div",{style:{marginLeft:"1rem"},children:[(0,i.jsx)("div",{style:{marginTop:".5rem",marginBottom:".5rem"},children:(0,i.jsx)(s.p,{children:"Disk usage information."})}),(0,i.jsx)(u(),{collapsible:!1,name:"total",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Total disk space in bytes.",example:5e11}}),(0,i.jsx)(u(),{collapsible:!1,name:"used",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Used disk space in bytes.",example:25e10}}),(0,i.jsx)(u(),{collapsible:!1,name:"free",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"Free disk space in bytes.",example:25e10}})]})]})})]})]})}),(0,i.jsx)(j.default,{label:"Example (from schema)",value:"Example (from schema)",children:(0,i.jsx)(p(),{responseExample:'{\n  "hostname": "my-linux-server",\n  "uptime": "0 days, 4 hours, 1 minute",\n  "load_average": {\n    "1min": 0.32,\n    "5min": 0.28,\n    "15min": 0.25\n  },\n  "memory": {\n    "total": 8388608,\n    "free": 2097152,\n    "used": 4194304\n  },\n  "disk": {\n    "total": 500000000000,\n    "used": 250000000000,\n    "free": 250000000000\n  }\n}',language:"json"})})]})})})})]}),(0,i.jsxs)(j.default,{label:"500",value:"500",children:[(0,i.jsx)("div",{children:(0,i.jsx)(s.p,{children:"Error retrieving system status"})}),(0,i.jsx)("div",{children:(0,i.jsx)(c(),{className:"openapi-tabs__mime",schemaType:"response",children:(0,i.jsx)(j.default,{label:"application/json",value:"application/json",children:(0,i.jsxs)(y(),{className:"openapi-tabs__schema",children:[(0,i.jsx)(j.default,{label:"Schema",value:"Schema",children:(0,i.jsxs)(a,{style:{},className:"openapi-markdown__details response","data-collapsed":!1,open:!0,children:[(0,i.jsx)("summary",{style:{},className:"openapi-markdown__details-summary-response",children:(0,i.jsx)("strong",{children:(0,i.jsx)(s.p,{children:"Schema"})})}),(0,i.jsx)("div",{style:{textAlign:"left",marginLeft:"1rem"}}),(0,i.jsxs)("ul",{style:{marginLeft:"1rem"},children:[(0,i.jsx)(u(),{collapsible:!1,name:"error",required:!0,schemaName:"string",qualifierMessage:void 0,schema:{type:"string",description:"A description of the error that occurred.",example:"Failed to retrieve the system status."}}),(0,i.jsx)(u(),{collapsible:!1,name:"details",required:!1,schemaName:"string",qualifierMessage:void 0,schema:{type:"string",description:"Additional details about the error, specifying which component failed.",example:"Failed to retrieve hostname."}}),(0,i.jsx)(u(),{collapsible:!1,name:"code",required:!0,schemaName:"integer",qualifierMessage:void 0,schema:{type:"integer",description:"The error code.",example:500}})]})]})}),(0,i.jsx)(j.default,{label:"Example (from schema)",value:"Example (from schema)",children:(0,i.jsx)(p(),{responseExample:'{\n  "error": "Failed to retrieve the system status.",\n  "details": "Failed to retrieve hostname.",\n  "code": 500\n}',language:"json"})})]})})})})]})]})})})]})}function k(e={}){const{wrapper:s}={...(0,t.R)(),...e.components};return s?(0,i.jsx)(s,{...e,children:(0,i.jsx)(M,{...e})}):M(e)}}}]);