resource "coolify_private_key" "example" {
  name        = "my-ssh-key"
  description = "SSH key for server access"
  private_key = file("~/.ssh/id_ed25519")
}