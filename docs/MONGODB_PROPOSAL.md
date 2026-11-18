# Assignment #6 — MongoDB Movie Recommendation Platform

## Project proposal

### Goal and scope

Yozigo is a small movie discovery and recommendation platform. A viewer can create an account, maintain a profile with favorite genres, browse and search a movie catalog, and record views, ratings, watchlist additions, and purchases. The API returns a simple personalized list using profile interests and interaction history. The implementation uses MongoDB as its primary database and Go's official MongoDB driver.

### Architecture

The React client calls a Go HTTP JSON API. The API validates requests, authenticates users with opaque server-side sessions, and reads/writes MongoDB collections. MongoDB stores users, movie catalog records, interaction events, and sessions. Indexes are created by the API at startup. The bundled seed inserts six synthetic movies into an empty catalog. The API can run alone against a local MongoDB or with Docker Compose.

### NoSQL model and trade-offs

The design uses separate collections for `users`, `movies`, `events`, and `sessions`. Users and movies are independent entities, while each event refers to one user and one movie by ObjectId. This avoids unbounded arrays in user documents: interaction history can grow without rewriting a large user record. It also supports event queries by user and date, and aggregate analytics by movie and event type. The trade-off is that a history response needs a second lookup to show full movie details; this first version returns event identifiers and types. Embedding favorite genres in the bounded user profile is appropriate because they are small and read with the profile.

### Collections

| Collection | Main fields | Access pattern |
|---|---|---|
| `users` | `_id`, `name`, normalized `email`, `passwordHash`, `favoriteGenres`, `createdAt` | Login by email; read/update own profile |
| `movies` | `_id`, `title`, `description`, `category`, `language`, `durationMinutes`, `rating`, `year`, `tags`, `posterUrl`, timestamps | Catalog listing, category filters, text search, CRUD |
| `events` | `_id`, `userId`, `movieId`, `type`, optional `value`, `createdAt` | Append interaction; read recent user history; compute preferences |
| `sessions` | `_id` (SHA-256 of opaque cookie token), `userId`, `expiresAt` | Resolve authenticated requests; automatic expiration |

`events.type` is one of `view`, `rating`, `watchlist`, or `purchase`; rating values are integers from 1 to 5. IDs are MongoDB ObjectIds. Passwords are stored as bcrypt hashes; raw session tokens are only held in HttpOnly cookies.

### Indexes and query speed

- `users.email` unique: constant-time email lookup and duplicate prevention.
- `movies` text index over title, description, category, and tags: full-text catalog search.
- `movies(category, title)`: category browsing with stable title order.
- `movies(rating, year desc)`: rating/year filtering and newest-first candidate order.
- `events(userId, createdAt desc)`: a viewer's recent history.
- `events(movieId, type)`: popularity and interaction analytics per movie.
- `events(userId, movieId, type)`: viewer-specific interaction lookup.
- `sessions.expiresAt` TTL index: MongoDB removes expired session documents.

### Search and recommendation method

Search accepts `q` for MongoDB text search and `category` for an exact case-insensitive category filter. The initial recommendation method is content-based and intentionally explainable: for a signed-in viewer, it filters out movies already present in the viewer's interaction history, optionally filters to their favorite genres, and returns up to 20 newest candidates. Anonymous requests receive a newest-first catalog list. This provides a working baseline for the final project; a later iteration can derive genre weights from positive ratings, watchlist additions, and completed purchases, then evaluate ranking quality with synthetic interaction data.

### API outline

| Method and path | Purpose |
|---|---|
| `POST /api/auth/register` or `/api/auth/signup` | Create user and session |
| `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me` | Authentication/session lifecycle |
| `PUT /api/users/me` | Update display name and favorite genres |
| `GET /api/movies`, `GET /api/movies/{id}` | Browse catalog and view a movie |
| `POST /api/admin/movies`, `PUT /api/admin/movies/{id}`, `DELETE /api/admin/movies/{id}` | Authenticated catalog CRUD |
| `GET /api/search?q=...&category=...` | Search by title, description, tags, and category |
| `POST /api/interactions` or `POST /api/movies/{id}/interactions` | Record view, rating, watchlist, or purchase |
| `GET /api/users/me/history` | Read the signed-in user's recent events |
| `GET /api/users/me/recommendations` or `/api/recommendations` | Retrieve recommendation candidates |
| `GET /api/health` | Check API and database connectivity |

All successful responses and errors are JSON. Authenticated writes use the session cookie created at registration/login.

### Test data and setup

The seed catalog contains ten demo movies across Sci-Fi, Comedy, Animation, Action, Thriller, and Adventure, with poster artwork and searchable tags. Start the stack from the repository root with `docker compose up --build`; the web app is served at `http://localhost:3000` and the API at `http://localhost:8080`. For local Go development, run MongoDB on port 27017 and start the API with `MONGODB_URI=mongodb://localhost:27017 MONGODB_DATABASE=yozigo`.

### Relevance to the dissertation

The dissertation concerns consistency models and event synchronization in scalable event-oriented RAG pipelines. This assignment is deliberately a smaller, independent recommendation application rather than an implementation of that research. Still, its interaction collection provides a concrete event stream: views, ratings, watchlist changes, and purchases are stored as append-only records with event time and entity identifiers. That structure can serve as a small experimental setting for later study of event ordering, duplicate delivery, and materialized user preference views. MongoDB's document model also makes the boundary between an event record and a derived recommendation profile visible. The current API writes events directly and computes recommendations from the stored profile and history; it does not claim to implement the dissertation's synchronization method or RAG pipeline. Results from the assignment can therefore be used as a prototype and teaching example for data modeling decisions, while research claims and synchronization experiments remain part of the dissertation work.

## Setup

From the project root:

```bash
docker compose up --build
```

The API creates required indexes on startup and seeds the catalog if it is empty. For direct API access, use `http://localhost:8080`; `GET /api/health` reports MongoDB status. Register an account with `POST /api/auth/register`, then send requests with the returned session cookie. After registration, set profile genres with `PUT /api/users/me`, for example `{"name":"Alex","favoriteGenres":["Sci-Fi","Animation"]}`.

Example event payload:

```json
{"movieId":"<movie ObjectId>","type":"rating","value":5}
```
