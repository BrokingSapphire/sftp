# synccli — desktop sync agent

Mirror a local folder to a Sapphire SFTP server over the REST API. Recursive,
incremental (SHA-256 diff), API-key auth. Re-uploading a changed file creates a
new version on the server.

## Build
```
go build -o synccli ./cmd/synccli
```

## Use
Generate an API key in the web app (API Keys), then:
```
# one-shot mirror
synccli -server http://localhost -key sftp_xxx -dir ~/Documents/work

# stay running, sync on change (Dropbox-style)
synccli -server http://localhost -key sftp_xxx -dir ~/work -watch
```
Env fallbacks: `SFTP_SERVER`, `SFTP_API_KEY`.

## Scope (v1)
Push mirror (local → server): uploads new/changed files, recreates the folder
tree, skips dotfiles. Two-way sync (pull, deletes, moves, conflict resolution)
is planned. Nothing is deleted on the server.
