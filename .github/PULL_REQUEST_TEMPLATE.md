## Summary

<!-- What changes and why. Link the issue if there is one. -->

## Lessons touched

<!-- Lesson ids such as go.basics.arrays-slices, or "none" for engine and docs changes. -->

## Checklist

- [ ] `python3 skills/tutor/scripts/tutor.py validate --strict` passes
- [ ] `python3 skills/tutor/scripts/tutor.py ci` passes (every solution runs against its tests, for every language that has one)
- [ ] `python3 -m unittest discover -s tests` passes (engine changes)
- [ ] Formatter-clean: `gofmt -l` prints nothing for Go; `ruff format --check` and `ruff check` pass for Python
- [ ] Tutor-only files (`TUTOR.md`, `quiz.json`, `solution/`, `snippets/`) are updated wherever the lesson changed
- [ ] Overlay files under `content/<lang>/shared/` or `content/<lang>/focus/` are updated for every language the lesson serves
- [ ] Paths in lesson prose use the `ada` example persona; nothing machine-specific
- [ ] Commit messages follow Conventional Commits
- [ ] I have read the [CLA](.github/CLA.md); the bot will ask me to sign on my first PR
