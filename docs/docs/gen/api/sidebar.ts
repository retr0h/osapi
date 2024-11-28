import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebar: SidebarsConfig = {
  apisidebar: [
    {
      type: "doc",
      id: "gen/api/osapi-a-crud-api-for-managing-linux-systems",
    },
    {
      type: "category",
      label: "Info",
      link: {
        type: "doc",
        id: "gen/api/osapi-a-crud-api-for-managing-linux-systems-info",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/retrieve-the-software-version",
          label: "Retrieve the software version",
          className: "api-method get",
        },
      ],
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
      ],
    },
    {
      type: "category",
      label: "System",
      link: {
        type: "doc",
        id: "gen/api/system-management-api-system-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/retrieve-system-hostname",
          label: "Retrieve system hostname",
          className: "api-method get",
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
    {
      type: "category",
      label: "Task",
      link: {
        type: "doc",
        id: "gen/api/task-api-task-operations",
      },
      items: [
        {
          type: "doc",
          id: "gen/api/add-an-task-item",
          label: "Add an task item",
          className: "api-method post",
        },
        {
          type: "doc",
          id: "gen/api/list-all-task-items",
          label: "List all task items",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/returns-the-total-number-of-task-items",
          label: "Returns the total number of task items",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/get-task-id",
          label: "Get a task item by ID",
          className: "api-method get",
        },
        {
          type: "doc",
          id: "gen/api/delete-task-id",
          label: "Delete a task item by ID",
          className: "api-method delete",
        },
      ],
    },
  ],
};

export default sidebar.apisidebar;
