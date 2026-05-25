# Publishing the Wiki

GitHub Wikis live in a separate Git repository named `<repo>.wiki.git`.

The wiki pages in this repository are kept under `wiki/` so they can be reviewed with normal code changes. A GitHub Actions workflow publishes those files into the GitHub Wiki repository.

## Automatic Publishing

The `wiki` workflow runs when changes under `wiki/` land on `main` or `master`. It can also be started manually from the Actions tab with `workflow_dispatch`.

The workflow syncs the contents of `wiki/` into:

```text
Justin-Arnold/p99.wiki.git
```

This keeps the project repository as the source of reviewable documentation while still using GitHub Wiki for the rendered pages.

## First-Time Setup on GitHub

Enable the GitHub Wiki for the repository before the first publish.

The workflow first tries to publish with the built-in GitHub Actions token. If GitHub rejects that token for wiki writes, create a repository secret named `WIKI_TOKEN` that contains a token with permission to write to the repository wiki. The workflow automatically prefers `WIKI_TOKEN` when it is present.

## Manual Fallback

Manual publishing should rarely be needed, but it is useful when debugging permissions.

```sh
git clone git@github.com:Justin-Arnold/p99.wiki.git p99.wiki
```

From the project checkout:

```sh
rsync -av --delete wiki/ ../p99.wiki/
cd ../p99.wiki
git status
git add .
git commit -m "Update p99 wiki"
git push
```

## Page Names

GitHub Wiki page names come from file names.

For example:

- `Home.md` becomes the Home page
- `Command-http.md` becomes a page named Command http
- `_Sidebar.md` becomes the wiki sidebar

## Review Rule

Keep the repository README short. Put deep explanations, command details, flag references, and conceptual walkthroughs in the wiki.
