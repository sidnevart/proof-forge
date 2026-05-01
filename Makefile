.PHONY: help bootstrap verify verify-skills list-skills docs tree

help:
	@printf "Available targets:\n"
	@printf "  bootstrap      Regenerate mirrored skills and scaffold inventories\n"
	@printf "  verify         Verify the skill system and scaffold structure\n"
	@printf "  verify-skills  Alias for verify\n"
	@printf "  list-skills    Print the expected skill inventory\n"
	@printf "  docs           List scaffolded docs\n"
	@printf "  tree           List scaffolded repository paths\n"

bootstrap:
	python3 scripts/bootstrap_skill_system.py

verify: verify-skills

verify-skills:
	python3 scripts/verify_skill_system.py

list-skills:
	python3 scripts/verify_skill_system.py --list

docs:
	find docs -maxdepth 3 -type f | sort

tree:
	find . -maxdepth 3 \( -path './.git' -o -path './__pycache__' \) -prune -o -type f -print | sort
