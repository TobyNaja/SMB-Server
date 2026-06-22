## What
<!-- one line describing what this PR does -->

## Why
<!-- requirement, issue number, or bug description -->

## Checklist
- [ ] `make test` passes
- [ ] `make lint-go` passes
- [ ] `make lint-fe` passes (if frontend changed)
- [ ] No `.env` or secrets in diff
- [ ] New mutating endpoints call `auditSvc.Log()`
- [ ] Conventional commit messages
- [ ] Architecture spec updated if structure changed
