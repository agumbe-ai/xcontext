# Optional Bash-output hook

`xcontext-opt-in.json` is intentionally not loaded by the plugin manifest. The normal and recommended integration has Claude call `xcontext_execute` before a noisy command runs.

The optional hook ingests successful Bash responses after execution. It does not remove the original response from Claude's context, so it provides preservation and redaction telemetry rather than token interception. Review the data-retention mode before enabling it, then copy its `PostToolUse` entry into your Claude Code hooks configuration.
