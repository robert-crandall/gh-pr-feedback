# VHS Demo Recording
# See https://github.com/charmbracelet/vhs for installation
# Run: vhs demo.tape

Output demo.gif

Set FontSize 16
Set Width 800
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 50ms

# Title
Type "# gh-pr-feedback - Collect and act on PR feedback"
Enter
Sleep 1s
Ctrl+L

# Show help
Type "gh pr-feedback --help"
Enter
Sleep 3s

# Summary mode demo
Ctrl+L
Type "# Summarize feedback from the current PR"
Enter
Sleep 500ms
Type "gh pr-feedback --action summary"
Enter
Sleep 4s

# Raw JSON mode
Ctrl+L
Type "# Get raw JSON for scripting"
Enter
Sleep 500ms
Type "gh pr-feedback --action raw | head -20"
Enter
Sleep 5s

# Apply mode teaser
Ctrl+L
Type "# Let Copilot CLI address the feedback"
Enter
Sleep 500ms
Type "gh pr-feedback --action apply"
Sleep 1s
Enter
Sleep 3s
Hide
Sleep 15s
Show
Sleep 3s
Ctrl+c 

# Wrap up
Ctrl+L
Type "# Install: gh extension install robert-crandall/gh-pr-feedback"
Enter
Sleep 2s
