---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "utils_consul_key Resource - utils"
subcategory: ""
description: |-
  This resource allows you to manage keys in Consul KV store.
---

# utils_consul_key (Resource)

This resource allows you to manage keys in Consul KV store.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `path` (String) The path to the key in the Consul KV store
- `value` (String) The value to set for the key in the Consul KV store

### Optional

- `delete` (Boolean) Whether to delete the key from the Consul KV store

### Read-Only

- `id` (String) The unique identifier for the exported service
