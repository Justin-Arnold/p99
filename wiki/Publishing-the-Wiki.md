# Publishing the Wiki

GitHub Wikis live in a separate Git repository named `<repo>.wiki.git`.

The wiki pages in this repository are kept under `wiki/` so they can be reviewed with normal code changes. To publish them, copy or sync those files into the GitHub Wiki repository.

## First-Time Setup

```sh
git clone git@github.com:justin/p99.wiki.git p99.wiki
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
