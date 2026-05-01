.PHONY: help bootstrap verify verify-skills list-skills docs tree verify-backend verify-web verify-docker verify-deploy

help:
	@printf "Available targets:\n"
	@printf "  bootstrap      Regenerate mirrored skills and scaffold inventories\n"
	@printf "  verify         Verify the skill system and scaffold structure\n"
	@printf "  verify-skills  Alias for verify\n"
	@printf "  verify-backend Run Go test suite\n"
	@printf "  verify-web     Run frontend lint, tests, and build\n"
	@printf "  verify-docker  Validate Docker image builds locally\n"
	@printf "  verify-deploy  Validate deploy compose and nginx assets exist\n"
	@printf "  list-skills    Print the expected skill inventory\n"
	@printf "  docs           List scaffolded docs\n"
	@printf "  tree           List scaffolded repository paths\n"

bootstrap:
	python3 scripts/bootstrap_skill_system.py

verify: verify-skills

verify-skills:
	python3 scripts/verify_skill_system.py

verify-backend:
	cd backend && go test ./...

verify-web:
	cd web && npm run lint && npm run test && npm run build

verify-docker:
	docker build -f backend/Dockerfile --build-arg TARGET=api .
	docker build -f backend/Dockerfile --build-arg TARGET=worker .
	docker build -f web/Dockerfile .

verify-deploy:
	test -f infra/docker/compose.prod.yml
	test -f infra/nginx/nginx.prod.high-port.conf
	test -f infra/docker/.env.prod.example
	test -f .github/workflows/ci.yml
	test -f .github/workflows/deploy.yml
	cp infra/docker/.env.prod.example infra/docker/.env.prod
	docker compose -f infra/docker/compose.prod.yml --env-file infra/docker/.env.prod.example config >/dev/null
	rm -f infra/docker/.env.prod

list-skills:
	python3 scripts/verify_skill_system.py --list

docs:
	find docs -maxdepth 3 -type f | sort

tree:
	find . -maxdepth 3 \( -path './.git' -o -path './__pycache__' \) -prune -o -type f -print | sort
