# k10s Demo - Dark Theme (Dracula)
Output assets/k10s-demo-dark.gif

Set Shell "zsh"
Set FontSize 13
Set Width 1400
Set Height 700
Set Theme "Dracula"
Set Padding 20
Set WindowBar Colorful
Set BorderRadius 8
Set TypingSpeed 80ms
Set Framerate 30

# Launch k10s
Type "k10s"
Enter
Wait+Screen /Running/
Sleep 500ms

# Scroll down through pods
Down 2
Sleep 300ms
Down 2
Sleep 300ms
Down 1
Sleep 600ms

# Press 'd' to open describe view for selected pod
Type "d"
Sleep 2s

# Jump to bottom of describe output with G
Type "G"
Sleep 600ms

# Toggle line numbers with 'n'
Type "n"
Sleep 800ms

# Go back to main pods view with Escape
Escape
Sleep 800ms

# Enter command mode, type :rs n, then Tab to autocomplete to 'nodes'
Type ":rs n"
Sleep 600ms
Tab
Sleep 600ms
Enter
Sleep 2s

# Quit
Type ":q"
Enter
Sleep 800ms
