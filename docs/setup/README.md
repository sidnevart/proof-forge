# Setup Docs

Здесь хранятся служебные документы для запуска, проверки и дальнейшего исполнения scaffold-а.

Основные документы:
- `implementation-prompts.md` — базовые следующие шаги для реализации
- `brand-foundation-prompts.md` — отдельные промпты для развития бренд-фундамента
- `recommended-order.md` — рекомендуемый порядок исполнения
- `setup-commands.md` — команды первичной настройки
- `verification-commands.md` — базовые команды проверки
- `skills-inventory.md` — список текущих skills
- `skills-paths.txt` — полный список путей к skill-файлам
- `files-inventory.txt` — инвентарь файлов scaffold-а

Как использовать:
- если нужен следующий шаг по продукту, начинать с `implementation-prompts.md`
- если нужно развивать brand/design foundation, начинать с `brand-foundation-prompts.md`
- перед крупной работой сверяться с `recommended-order.md`
- после изменений прогонять команды из `verification-commands.md`

Короткая проверка для brand foundation:

```bash
find docs/brand -maxdepth 1 -type f | sort
find docs/product -maxdepth 1 -type f | sort
find docs/architecture -maxdepth 1 -type f | sort
sed -n '1,260p' AGENTS.md
sed -n '1,260p' docs/setup/implementation-prompts.md
```
