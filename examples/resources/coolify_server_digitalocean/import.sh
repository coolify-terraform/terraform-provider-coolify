#!/bin/sh
# Import requires Coolify >= v4.2.0. DigitalOcean create-only fields are empty after import.
# NOTE: Coolify GET often omits private_key_uuid, so import leaves it empty.
# Set the same UUID that is already on the server in your .tf config
# BEFORE running terraform plan, or the next apply will PATCH a new SSH key.
terraform import coolify_server_digitalocean.app <server-uuid>
