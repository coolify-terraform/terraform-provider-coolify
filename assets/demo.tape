# VHS tape for terraform-provider-coolify README demo.
# Re-record:  vhs assets/demo.tape
# Reference:  https://github.com/charmbracelet/vhs

Output assets/demo.gif

Set Shell "bash"
Set FontFamily "JetBrains Mono"
Set FontSize 16
Set Width 900
Set Height 500
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 40ms

# ── Setup (hidden) ──────────────────────────────────────────
Hide
Type "export PS1='$ '"
Enter
Sleep 200ms
Type "cd /tmp/coolify-demo"
Enter
Sleep 200ms
Type "export PATH=/tmp/coolify-demo/bin:$PATH"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 200ms
Show
Sleep 500ms

# ── Show the Terraform config ──────────────────────────────
Type "cat main.tf"
Enter
Sleep 2500ms

# ── Initialize the provider ────────────────────────────────
Type "terraform init"
Enter
Sleep 4s

# ── Apply the configuration ────────────────────────────────
Type "terraform apply -auto-approve"
Enter
Sleep 8s
