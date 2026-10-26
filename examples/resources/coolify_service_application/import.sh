#!/bin/sh
# Import id is service_uuid/uuid. Read the application UUID from
# coolify_service_applications. Destroy removes Terraform state only.
# The compose container and its URL stay in Coolify.
terraform import coolify_service_application.web <service-uuid>/<application-uuid>
