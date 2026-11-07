# WASAText

A full-stack messaging web app, similar to WhatsApp or Telegram, built for the
**Web and Software Architecture** course at Sapienza University of Rome (2025).
Users can chat one-on-one or in groups, exchange text and photos, reply, forward,
react to messages and track delivery with read receipts.

The project covers the full development cycle required by the course: API design
with OpenAPI, a Go backend, a Vue.js single-page frontend and Docker deployment.


## Features

**Messaging**
- One-on-one and group conversations, listed in reverse chronological order with
  the latest message preview, timestamp and profile/group photo
- Text and photo messages (PNG, JPG, GIF and other image formats)
- Reply to a specific message, shown as a quote inside the bubble
- Forward one or more messages to any user or group, with a "Forwarded" label
- Emoji reactions, visible to everyone in the conversation
- Delete messages, for yourself or for everyone if you are the sender
- Read receipts: ✓ received by all recipients, ✓✓ read by all recipients
- Automatic refresh of conversations and messages

**Groups**
- Create a group with any number of users
- Add new members (users cannot join groups on their own); new members can read
  the previous conversation history
- Rename the group, change its photo, or leave it

**Users**
- Simplified login with just a username: new users are registered automatically
- Search other users by username and start a new chat
- Change your username (must be unique) and your profile photo

**Extras beyond the course requirements**
- Starred messages, per chat and in a global "Saved messages" list
- Photo gallery for each conversation
- Full-screen image viewer with download
- In-chat message search

## Tech stack

| Layer | Technologies |
|---|---|
| API design | OpenAPI 3 (`doc/api.yaml`) |
| Backend | Go, REST API, SQL database |
| Frontend | Vue.js (single-page application), Axios, Bootstrap |
| Deployment | Docker (separate backend and frontend images), Docker Compose, Nginx |

Authentication follows the course's simplified login: the backend returns a user
identifier, which the frontend sends as a Bearer token in the `Authorization`
header of every request.

## API overview

The full specification is in [`doc/api.yaml`](doc/api.yaml). Main endpoints:

| Method | Endpoint | Description |
|---|---|---|
| POST | `/session` | Log in (or register) with a username |
| GET | `/users` | Search users by username |
| PUT | `/user/name`, `/user/photo` | Update own username / photo |
| GET, POST | `/conversations` | List conversations / start a new one |
| GET | `/conversations/{conversationId}` | Get the messages of a conversation |
| POST | `/conversations/{conversationId}/messages` | Send a message |
| POST | `/messages/forwarded` | Forward messages |
| PUT, DELETE | `/messages/{messageId}/reactions/me` | Add / remove a reaction |
| DELETE | `/messages/{messageId}` | Delete a message |
| POST | `/groups` | Create a group |
| POST | `/groups/{groupId}/participants` | Add a member |
| PUT | `/groups/{groupId}/name`, `/groups/{groupId}/photo` | Update group name / photo |
| DELETE | `/user/groups/{groupId}` | Leave a group |

## Project structure

```
cmd/webapi/           backend entry point
service/api/          HTTP handlers, one file per operation
service/database/     database interface and queries
webui/src/views/      main pages (login, chat, profile)
webui/src/components/ reusable components (message item, modals, image viewer)
doc/api.yaml          OpenAPI specification
Dockerfile.backend    builds the Go web API (port 3000)
Dockerfile.frontend   builds the Vue app and serves it with Nginx (port 80)
docker-compose.yml    runs both, with a shared volume for uploaded images
```

## Running the app

With Docker:

```shell
docker compose up --build
```

Then open http://localhost in your browser. To try the app, log in with different
usernames in separate browsers (or private windows) and start chatting.

### Development mode

Backend:

```shell
go run ./cmd/webapi/
```

Frontend (in a second terminal):

```shell
./open-node.sh
# inside the container
yarn run dev
```

## Credits

Built on the [Fantastic Coffee (decaffeinated)](https://github.com/sapienzaapps/fantastic-coffee-decaffeinated)
template by Enrico Bassetti (MIT License). Project specifications by Emanuele Panizzi
and Enrico Bassetti for the WASA course, Sapienza University of Rome.

## License

See [LICENSE](LICENSE).
