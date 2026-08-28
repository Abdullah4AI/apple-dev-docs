# Game Center relationship replacement confirmation

Game Center relationship setters now accept an experimental `--confirm` flag.
Invocations that omit it continue to work in the 4.x compatibility window and
emit a deprecation warning. Starting in 5.0.0, `--confirm` will be required to
acknowledge that these setters replace the complete relationship collection and
may remove existing members.
