# Import an environment variable on an application:
terraform import coolify_environment_variable.example application:<application-uuid>:<env-var-uuid>

# Import an environment variable on a service:
terraform import coolify_environment_variable.example service:<service-uuid>:<env-var-uuid>