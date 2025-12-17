# 2025-12-17 00:10 - c2p.sh fix

- Actions: Removed stray `:w` from `c2p.sh` (fixes invalid `--path` value), ran `./c2p.sh` to verify no error.
- Decisions: Treat `kagent-adk-agent:w` as a typo and change to `kagent-adk-agent`.
- Next steps: Commit the change (`git add c2p.sh && git commit -m "fix: remove stray :w from c2p.sh"`) and push if desired; run `./c2p.sh` locally to confirm behavior.
- Lessons/insights: Likely a stray editor command or paste added `:w`; keep small scripts simple and review before running.
