#!/bin/sh
# Requires Coolify >= v4.5.0.
# Purchase fields are not returned by GET /servers/{uuid}. After import they
# are empty. Set cloud_provider_token_uuid, item_id, data_center_id,
# template_id, public_key_ids, post_install_script_id, and enable_backups in
# configuration to the values used at purchase. Terraform adopts them and
# does not buy another VPS. Changing a value already in state replaces the
# server and purchases again.
# Coolify often omits private_key_uuid as well. Set the UUID that is already
# on the server before planning.
# Import does not cancel the Hostinger VPS. Destroy deletes only the Coolify
# server record. The subscription keeps billing in hPanel.
terraform import coolify_server_hostinger.app <server-uuid>
