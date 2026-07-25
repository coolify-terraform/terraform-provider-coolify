#!/bin/sh
# Import requires Coolify >= v4.2.0. DigitalOcean create-only fields are empty after import.
terraform import coolify_server_digitalocean.app <server-uuid>
