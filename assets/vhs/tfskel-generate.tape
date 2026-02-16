Output "~/workspaces/personal/github/tfskel/assets/tfskel-generate.gif"

Require echo
Set Margin 10
Set MarginFill "#6B50FF"
Set Padding 10

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 600
Set PlaybackSpeed 0.7

Type "echo 'Welcome to tfskel!'" Sleep 500ms  Enter
Type "tfskel init" Sleep 500ms  Enter
Sleep 2s
Type "tfskel generate test --env dev --region eu-central-1 --create-github-workflows" Sleep 500ms  Enter
Sleep 5s
Type "clear" Sleep 300ms Enter
Sleep 500ms
Type "tree -a" Sleep 500ms  Enter
Sleep 5s
