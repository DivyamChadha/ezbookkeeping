# Fork notes (branch `custom`)

This is a personal fork of mayswind/ezbookkeeping. `custom` is the branch that gets built and deployed. Read this before changing anything. If a local `.memory/` folder exists (it is gitignored and never pushed), read `.memory/MEMORY.md` too: it holds the private deployment details and the project history.

## Build and deploy

- Pushing to `custom` runs `.github/workflows/build-custom.yml`, which builds the image and pushes the `:custom` tag to ghcr. The owner's deployment pulls that tag automatically, so **a push to `custom` goes live**. Never push or merge to `custom` without the owner's explicit go-ahead.
- Back up the database before any data or schema change.

## Working rules

- Develop on a feature branch and test against a **copy** of the database on another port, never against live data.
- Squash a feature into one commit on `custom` before pushing.
- Prefer building a feature over a manual workaround. Before building, explain in plain words what each action does to the data and which files change, then wait for a go-ahead.
- Keep fork code in **new files**; make only small, appended edits to upstream files (listed below), so upstream merges stay conflict-free.
- Do not commit personal information (hostnames, paths, account or category names from real data, tokens) to this public repository.

## What the fork adds

1. **CI**: `.github/workflows/build-custom.yml` (new) and a small change to `build-non-main-branch.yml` (skip the multi-platform validation build on `custom`).
2. **Split transactions** (upstream rejects these by design, see mayswind/ezbookkeeping issue #325). No schema change: the original transaction is edited in place, extra rows are added, nothing is deleted.
   - Backend: `POST /api/v1/transactions/split.json` in `pkg/api/transactions_split.go`, `pkg/services/transactions_split.go`, `pkg/models/transaction_split.go`, `pkg/errs/transaction_split.go`, with tests in `pkg/services/transactions_split_test.go`. Request: `{id, categoryId, amount (stays on the original), comment?, expenseItems:[{categoryId, amount, comment?}], transferItems:[{categoryId, destinationAccountId, amount, destinationAmount?, comment?}]}`. The parts must add up to the original amount.
   - Four actions: *Split by category* (expense into several expenses), *Split with people* (expense into the user's share plus a transfer of the rest to a receivable account), *Mark as repayment* (income into a transfer from the receivable account), *Mark as refund* (income into a negative expense in a chosen category). Repayment and refund are plain calls to the existing `modify.json`.
   - One shared Receivables account named "Due back", created on first use with zero balance. The split transfer category defaults to a transfer sub-category named "Due back" if one exists, otherwise the first visible transfer sub-category.
   - "Split with people" asks what the others owe in total (default is an equal split rounded to whole currency units); the user's share absorbs the remainder. The comment note is a short prefix, and the original comment is trimmed to fit 255 characters.
   - Desktop UI: `src/views/desktop/transactions/list/dialogs/SplitActionsButton.vue` and `SplitDialog.vue`. Mobile UI: `src/views/mobile/transactions/split/`. Shared logic: `src/views/base/transactions/TransactionSplitDialogBase.ts`. Pure helpers and vitest tests: `src/lib/transaction_split.ts`, `src/lib/__tests__/transaction_split.test.ts`.

## Upstream files the fork edits (check these first on every upstream merge)

- `cmd/webserver.go`: one route line for `split.json`.
- `src/locales/en.json`: two blocks of split strings, inserted after stable lines. Other locales fall back to English.
- `src/views/desktop/transactions/list/dialogs/EditDialog.vue`: an import, one tag, and `onSplitDone`.
- `src/views/mobile/transactions/EditPage.vue`: an actions group in the "⋯" action sheet.
- `.github/workflows/build-non-main-branch.yml`.

## Syncing with upstream

- Remotes: `origin` is the fork, `upstream` is mayswind/ezbookkeeping (default branch `main`, releases are tags such as `v2.0.1`).
- Procedure: `git fetch upstream`, create a branch from `custom`, `git merge upstream/main` (a merge, not a rebase, to match the existing history), resolve conflicts in the files above, run the checks below, and test on a database copy before pushing.
- Re-run the split tests whenever upstream changes: the `Transaction` model or tables (the split service uses them directly), the `transactions` APIs (`modify.json` is relied on for repayment and refund), the transaction category store, the `AccountCategory` / `CategoryType` enums, or the Framework7 setup in `mobile-main.ts` (the split popup depends on which F7 modules are registered).
- Check whether upstream has added its own split or repayment feature; if so, consider dropping the fork version.
- Upstream schema migrations run automatically on start, so back up first.

## Checks

- Go: `go test ./pkg/services/ -run 'TestSplitTransaction|TestModifyTransaction_Mark'`. Build: `CGO_ENABLED=1 go build -trimpath -o ezbookkeeping ezbookkeeping.go`.
- Frontend: `npx vue-tsc --noEmit` (about 2 minutes), `npx eslint <files>` (do NOT run `npm run lint`, it runs `eslint . --fix` on the whole repo), `npx vitest run src/lib/__tests__/transaction_split.test.ts`. `npm run build` is slow on small machines (about 9 minutes on a Raspberry Pi).
- After a data-touching change, run the `userdata transaction-check` command against a database copy.
- UI changes cannot be verified visually in a headless environment; have the owner try them on a test instance.

## Gotchas learned

- Framework7: custom components inside `f7-list` must go in the `#list` slot. The default slot's render calls `.indexOf()` on each child's component `name`, and `<script setup>` components only have `__name` in production builds, so the list silently renders empty.
- `list-number-input` clamps to min/max on every keystroke, which makes typing unusable, and `f7-stepper` is not loaded (`mobile-main.ts` registers a fixed list of F7 modules). The split popup uses plain `f7-link` minus/plus buttons.
- The category API refuses to move a sub-category across types, so a category with a parent of the wrong type needs a direct database fix. Default pickers only choose sub-categories whose own type matches.

## Ideas for later

- A rule in an external importer that marks incoming repayments by sender name automatically.
- Remember the last-used category in the split popups.
- Per-friend tracking of what is owed (currently one shared receivable account, by design).
- Translations of the split strings for other locales.
