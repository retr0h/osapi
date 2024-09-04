import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "gen/api/osapi-a-crud-api-for-managing-linux-systems",
    },
    {
      type: "category",
      label: "Ping",
      link: {
        type: "doc",
        id: "gen/api/minimal-ping-api-endpoint-ping-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/retrieve-ping-status",
          label: "Retrieve ping status",
          className: "api-method get",
        },
      ],
    },
    {
      type: "category",
      label: "System",
      link: {
        type: "doc",
        id: "gen/api/system-status-api-system-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/retrieve-system-status",
          label: "Retrieve system status",
          className: "api-method get",
        },
      ],
    },
  ],
};

export default sidebar.apisidebar;
