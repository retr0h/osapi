import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "gen/api/osapi-a-crud-api-for-managing-linux-systems",
    },
    {
      type: "category",
      label: "Network",
      link: {
        type: "doc",
        id: "gen/api/network-management-api-network-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/ping-a-remote-server",
          label: "Ping a remote server",
          className: "api-method post",
        },
      ],
    },
    {
      type: "category",
      label: "Network/DNS",
      link: {
        type: "doc",
        id: "gen/api/network-management-api-dns-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/get-network-dns",
          label: "List DNS servers",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/put-network-dns",
          label: "Update DNS servers",
          className: "api-method put",
        },
        {
          type: "doc",
          id: "gen/api/delete-network-dns-server-id",
          label: "Delete a DNS server",
          className: "api-method delete",
        },
      ],
    },
    {
      type: "category",
      label: "Queue",
      link: {
        type: "doc",
        id: "gen/api/queue-api-queue-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/add-an-item-to-the-queue",
          label: "Add an item to the queue",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "gen/api/list-all-queue-items",
          label: "List all queue items",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/get-queue-id",
          label: "Get a queue item by ID",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/delete-queue-id",
          label: "Delete a queue item by ID",
          className: "api-method delete",
        },
      ],
    },
    {
      type: "category",
      label: "System/Status",
      link: {
        type: "doc",
        id: "gen/api/system-management-api-system-status",
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
