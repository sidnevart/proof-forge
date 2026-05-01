# Verification Commands

```bash
python3 scripts/verify_skill_system.py
make verify
find .claude/skills -mindepth 1 -maxdepth 1 -type d | wc -l
find .codex/skills -mindepth 1 -maxdepth 1 -type d | wc -l
rg --files .claude/skills .codex/skills | rg 'SKILL.md$' | wc -l
```
