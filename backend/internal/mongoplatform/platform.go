// Package mongoplatform implements the recommendation platform API on MongoDB.
package mongoplatform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type Platform struct {
	Client *mongo.Client
	DB     *mongo.Database
}
type User struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name           string        `bson:"name" json:"name"`
	Role           string        `bson:"role" json:"role"`
	Email          string        `bson:"email" json:"email"`
	Hash           string        `bson:"passwordHash" json:"-"`
	FavoriteGenres []string      `bson:"favoriteGenres,omitempty" json:"favoriteGenres,omitempty"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
}
type Movie struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string        `bson:"title" json:"title"`
	Description     string        `bson:"description" json:"description"`
	Category        string        `bson:"category" json:"category"`
	Language        string        `bson:"originalLanguage" json:"language"`
	DurationMinutes int           `bson:"durationMinutes" json:"durationMinutes"`
	Rating          string        `bson:"rating" json:"rating"`
	Year            int           `bson:"year,omitempty" json:"year,omitempty"`
	Tags            []string      `bson:"tags,omitempty" json:"tags,omitempty"`
	PosterURL       *string       `bson:"posterUrl,omitempty" json:"posterUrl,omitempty"`
	Source          string        `bson:"source,omitempty" json:"-"`
	SourceID        string        `bson:"sourceId,omitempty" json:"-"`
	CreatedAt       time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time     `bson:"updatedAt" json:"updatedAt"`
}
type Event struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     bson.ObjectID `bson:"userId" json:"userId"`
	MovieID    bson.ObjectID `bson:"movieId" json:"movieId"`
	MovieTitle string        `bson:"movieTitle,omitempty" json:"movieTitle,omitempty"`
	Type       string        `bson:"type" json:"type"`
	Value      float64       `bson:"value,omitempty" json:"value,omitempty"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
	Source     string        `bson:"source,omitempty" json:"-"`
	SourceID   string        `bson:"sourceId,omitempty" json:"-"`
}

func Connect(ctx context.Context, uri, dbname string) (*Platform, error) {
	c, e := mongo.Connect(options.Client().ApplyURI(uri))
	if e != nil {
		return nil, e
	}
	if e = c.Ping(ctx, nil); e != nil {
		return nil, e
	}
	p := &Platform{c, c.Database(dbname)}
	return p, p.Indexes(ctx)
}
func (p *Platform) Indexes(ctx context.Context) error {
	_, e := p.DB.Collection("users").Indexes().CreateMany(ctx, []mongo.IndexModel{{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)}})
	if e != nil {
		return e
	}
	_, e = p.DB.Collection("movies").Indexes().CreateMany(ctx, []mongo.IndexModel{{Keys: bson.D{{Key: "title", Value: "text"}, {Key: "description", Value: "text"}, {Key: "category", Value: "text"}, {Key: "tags", Value: "text"}}, Options: options.Index().SetName("movie_text_search")}, {Keys: bson.D{{Key: "title", Value: 1}}}, {Keys: bson.D{{Key: "category", Value: 1}, {Key: "title", Value: 1}}}, {Keys: bson.D{{Key: "rating", Value: 1}, {Key: "year", Value: -1}}}, {Keys: bson.D{{Key: "year", Value: -1}, {Key: "title", Value: 1}}}})
	if e != nil {
		return e
	}
	// Replace the simple chronological index with a covering index for the
	// recommendation scan. This lets MongoDB satisfy both sort and projection
	// without fetching each event document.
	_ = p.DB.Collection("events").Indexes().DropOne(ctx, "createdAt_-1")
	_, e = p.DB.Collection("events").Indexes().CreateMany(ctx, []mongo.IndexModel{{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}}, {Keys: bson.D{{Key: "createdAt", Value: -1}, {Key: "userId", Value: 1}, {Key: "movieId", Value: 1}, {Key: "type", Value: 1}, {Key: "value", Value: 1}}, Options: options.Index().SetName("recommendation_event_scan")}, {Keys: bson.D{{Key: "movieId", Value: 1}, {Key: "type", Value: 1}}}, {Keys: bson.D{{Key: "userId", Value: 1}, {Key: "movieId", Value: 1}, {Key: "type", Value: 1}}}})
	if e != nil {
		return e
	}
	_, e = p.DB.Collection("sessions").Indexes().CreateMany(ctx, []mongo.IndexModel{{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}})
	return e
}
func (p *Platform) Seed(ctx context.Context) error {
	c := p.DB.Collection("movies")
	now := time.Now().UTC()
	data := []any{
		Movie{Title: "Interstellar", Description: "A team travels beyond this galaxy to discover whether mankind has a future among the stars.", Category: "Sci-Fi", Language: "English", DurationMinutes: 169, Rating: "UA", Year: 2014, Tags: []string{"space", "drama", "adventure"}, PosterURL: poster("gEU2QniE6E77NI6lCU6MxlNBvIx.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Inception", Description: "A skilled thief enters dreams to steal secrets and faces one final challenge.", Category: "Sci-Fi", Language: "English", DurationMinutes: 148, Rating: "UA", Year: 2010, Tags: []string{"dreams", "thriller", "action"}, PosterURL: poster("oYuLEt3zVCKq57qu2F8dT7NIa6f.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "The Grand Budapest Hotel", Description: "A concierge and his lobby boy become involved in a priceless painting and family fortune.", Category: "Comedy", Language: "English", DurationMinutes: 99, Rating: "A", Year: 2014, Tags: []string{"comedy", "adventure"}, PosterURL: poster("eWdyYQreja6JGCzqHWXpWHDrrPo.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Spirited Away", Description: "A young girl enters a world ruled by spirits and must find a way to return home.", Category: "Animation", Language: "Japanese", DurationMinutes: 125, Rating: "U", Year: 2001, Tags: []string{"fantasy", "family", "animation"}, PosterURL: poster("39wmItIWsg5sZMyRUHLkWBcuVCM.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "The Dark Knight", Description: "Batman faces a criminal mastermind who pushes Gotham into chaos.", Category: "Action", Language: "English", DurationMinutes: 152, Rating: "UA", Year: 2008, Tags: []string{"action", "crime", "drama"}, PosterURL: poster("qJ2tW6WMUDux911r6m7haRef0WH.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Arrival", Description: "A linguist works to communicate with visitors whose arrival changes humanity's understanding of time.", Category: "Sci-Fi", Language: "English", DurationMinutes: 116, Rating: "UA", Year: 2016, Tags: []string{"aliens", "drama", "mystery"}, PosterURL: poster("x2FJsf1ElAgr63Y3PNPtJrcmpoe.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Parasite", Description: "A poor family gradually enters the lives of a wealthy household, with unexpected consequences.", Category: "Thriller", Language: "Korean", DurationMinutes: 132, Rating: "A", Year: 2019, Tags: []string{"thriller", "drama", "class"}, PosterURL: poster("7IiTTgloJzvGI1TAYymCfbfl3vT.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Dune", Description: "Paul Atreides travels to a dangerous desert planet central to humanity's future.", Category: "Sci-Fi", Language: "English", DurationMinutes: 155, Rating: "UA", Year: 2021, Tags: []string{"space", "adventure", "drama"}, PosterURL: poster("d5NXSklXo0qyIYkgV94XAgMIckC.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "The Matrix", Description: "A computer hacker discovers the reality he knows is a simulation and joins a rebellion.", Category: "Sci-Fi", Language: "English", DurationMinutes: 136, Rating: "UA", Year: 1999, Tags: []string{"sci-fi", "action", "cyberpunk"}, PosterURL: poster("f89U3ADr1oiB1s9GkdPOEpXUk5H.jpg"), CreatedAt: now, UpdatedAt: now},
		Movie{Title: "Everything Everywhere All at Once", Description: "An exhausted laundromat owner is pulled into an adventure across parallel universes.", Category: "Adventure", Language: "English", DurationMinutes: 140, Rating: "A", Year: 2022, Tags: []string{"multiverse", "comedy", "action"}, PosterURL: poster("w3LxiVYdWWRvEVdn5RYq6jIqkb1.jpg"), CreatedAt: now, UpdatedAt: now},
	}
	for _, item := range data {
		m := item.(Movie)
		insert := m
		insert.PosterURL = nil
		_, e := c.UpdateOne(ctx, bson.M{"title": m.Title}, bson.M{"$set": bson.M{"posterUrl": m.PosterURL}, "$setOnInsert": insert}, options.UpdateOne().SetUpsert(true))
		if e != nil {
			return e
		}
	}
	return nil
}
func poster(path string) *string { u := "https://image.tmdb.org/t/p/w500/" + path; return &u }
func write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func body(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}
func oid(s string) (bson.ObjectID, error) { return bson.ObjectIDFromHex(s) }
func (p *Platform) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if p.Client.Ping(r.Context(), nil) != nil {
			write(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ok", "database": "MongoDB"})
	})
	m.HandleFunc("POST /api/auth/register", p.register)
	m.HandleFunc("POST /api/auth/signup", p.register)
	m.HandleFunc("POST /api/auth/login", p.login)
	m.HandleFunc("POST /api/auth/logout", p.logout)
	m.HandleFunc("GET /api/auth/me", p.me)
	m.HandleFunc("PUT /api/users/me", p.updateProfile)
	m.HandleFunc("GET /api/movies", p.list)
	m.HandleFunc("GET /api/movies/{id}", p.getMovie)
	m.HandleFunc("POST /api/admin/movies", p.create)
	m.HandleFunc("PUT /api/admin/movies/{id}", p.update)
	m.HandleFunc("DELETE /api/admin/movies/{id}", p.delete)
	m.HandleFunc("GET /api/search", p.search)
	m.HandleFunc("GET /api/movies/search", p.search)
	m.HandleFunc("POST /api/interactions", p.event)
	m.HandleFunc("POST /api/movies/{id}/interactions", p.eventForMovie)
	m.HandleFunc("GET /api/users/me/history", p.history)
	m.HandleFunc("GET /api/users/me/recommendations", p.recommend)
	m.HandleFunc("GET /api/recommendations", p.recommend)
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { write(w, 404, map[string]string{"error": "Not found"}) })
	return m
}
func (p *Platform) register(w http.ResponseWriter, r *http.Request) {
	var x struct{ Name, Email, Password string }
	if body(r, &x) != nil || strings.TrimSpace(x.Name) == "" || len(x.Password) < 8 {
		write(w, 400, map[string]string{"error": "name, email, and password (8+ chars) required"})
		return
	}
	x.Email = strings.ToLower(strings.TrimSpace(x.Email))
	hash, e := bcrypt.GenerateFromPassword([]byte(x.Password), bcrypt.DefaultCost)
	if e != nil {
		write(w, 500, nil)
		return
	}
	u := User{Name: x.Name, Email: x.Email, Role: "customer", Hash: string(hash), CreatedAt: time.Now().UTC()}
	res, e := p.DB.Collection("users").InsertOne(r.Context(), u)
	if mongo.IsDuplicateKeyError(e) {
		write(w, 409, map[string]string{"error": "email already registered"})
		return
	}
	if e != nil {
		write(w, 500, map[string]string{"error": e.Error()})
		return
	}
	u.ID = res.InsertedID.(bson.ObjectID)
	p.session(w, r, u)
	write(w, 201, map[string]any{"user": u})
}
func (p *Platform) login(w http.ResponseWriter, r *http.Request) {
	var x struct{ Email, Password string }
	if body(r, &x) != nil {
		write(w, 400, nil)
		return
	}
	var u User
	e := p.DB.Collection("users").FindOne(r.Context(), bson.M{"email": strings.ToLower(strings.TrimSpace(x.Email))}).Decode(&u)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(x.Password)) != nil {
		write(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}
	p.session(w, r, u)
	write(w, 200, map[string]any{"user": u})
}
func (p *Platform) session(w http.ResponseWriter, r *http.Request, u User) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	expiry := time.Now().Add(24 * time.Hour)
	_, _ = p.DB.Collection("sessions").InsertOne(r.Context(), bson.M{"_id": hex.EncodeToString(sum[:]), "userId": u.ID, "expiresAt": expiry})
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: expiry})
}
func (p *Platform) current(r *http.Request) (User, bool) {
	c, e := r.Cookie("session")
	if e != nil {
		return User{}, false
	}
	s := sha256.Sum256([]byte(c.Value))
	var doc struct {
		UserID bson.ObjectID `bson:"userId"`
	}
	if p.DB.Collection("sessions").FindOne(r.Context(), bson.M{"_id": hex.EncodeToString(s[:]), "expiresAt": bson.M{"$gt": time.Now()}}).Decode(&doc) != nil {
		return User{}, false
	}
	var u User
	if p.DB.Collection("users").FindOne(r.Context(), bson.M{"_id": doc.UserID}).Decode(&u) != nil {
		return User{}, false
	}
	return u, true
}
func (p *Platform) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("session"); e == nil {
		s := sha256.Sum256([]byte(c.Value))
		_, _ = p.DB.Collection("sessions").DeleteOne(r.Context(), bson.M{"_id": hex.EncodeToString(s[:])})
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	write(w, 200, map[string]string{"status": "logged out"})
}
func (p *Platform) me(w http.ResponseWriter, r *http.Request) {
	u, ok := p.current(r)
	if !ok {
		write(w, 200, map[string]any{"user": nil})
		return
	}
	write(w, 200, map[string]any{"user": u})
}
func (p *Platform) updateProfile(w http.ResponseWriter, r *http.Request) {
	u, ok := p.current(r)
	if !ok {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var x struct {
		Name           string   `json:"name"`
		FavoriteGenres []string `json:"favoriteGenres"`
	}
	if body(r, &x) != nil || strings.TrimSpace(x.Name) == "" {
		write(w, 400, map[string]string{"error": "name is required"})
		return
	}
	_, err := p.DB.Collection("users").UpdateOne(r.Context(), bson.M{"_id": u.ID}, bson.M{"$set": bson.M{"name": strings.TrimSpace(x.Name), "favoriteGenres": x.FavoriteGenres}})
	if err != nil {
		write(w, 500, nil)
		return
	}
	u.Name = strings.TrimSpace(x.Name)
	u.FavoriteGenres = x.FavoriteGenres
	write(w, 200, map[string]any{"user": u})
}
func (p *Platform) list(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	if c := r.URL.Query().Get("category"); c != "" {
		filter["category"] = bson.M{"$regex": "^" + c + "$", "$options": "i"}
	}
	pageSize := 40
	if requested, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && requested > 0 && requested <= 100 {
		pageSize = requested
	}
	pageSkip := 0
	if requested, err := strconv.Atoi(r.URL.Query().Get("skip")); err == nil && requested > 0 {
		pageSkip = requested
	}
	cur, e := p.DB.Collection("movies").Find(r.Context(), filter, options.Find().SetSort(bson.D{{Key: "year", Value: -1}, {Key: "title", Value: 1}}).SetSkip(int64(pageSkip)).SetLimit(int64(pageSize)))
	if e != nil {
		write(w, 500, nil)
		return
	}
	defer cur.Close(r.Context())
	var a []Movie
	if cur.All(r.Context(), &a) != nil {
		write(w, 500, nil)
		return
	}
	if a == nil {
		a = []Movie{}
	}
	write(w, 200, map[string]any{"movies": a})
}
func (p *Platform) getMovie(w http.ResponseWriter, r *http.Request) {
	id, e := oid(r.PathValue("id"))
	if e != nil {
		write(w, 400, nil)
		return
	}
	var x Movie
	e = p.DB.Collection("movies").FindOne(r.Context(), bson.M{"_id": id}).Decode(&x)
	if e != nil {
		write(w, 404, map[string]string{"error": "movie not found"})
		return
	}
	write(w, 200, map[string]any{"movie": x, "shows": []any{}})
}
func (p *Platform) create(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.current(r); !ok {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var x Movie
	if body(r, &x) != nil || strings.TrimSpace(x.Title) == "" {
		write(w, 400, map[string]string{"error": "title required"})
		return
	}
	x.Title = strings.TrimSpace(x.Title)
	if x.Category == "" {
		x.Category = "Other"
	}
	if x.Language == "" {
		x.Language = "English"
	}
	if x.Rating == "" {
		x.Rating = "UA"
	}
	x.CreatedAt = time.Now().UTC()
	x.UpdatedAt = x.CreatedAt
	res, e := p.DB.Collection("movies").InsertOne(r.Context(), x)
	if e != nil {
		write(w, 500, nil)
		return
	}
	x.ID = res.InsertedID.(bson.ObjectID)
	write(w, 201, map[string]any{"movie": x})
}
func (p *Platform) update(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.current(r); !ok {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id, e := oid(r.PathValue("id"))
	var x Movie
	if e != nil || body(r, &x) != nil {
		write(w, 400, nil)
		return
	}
	x.UpdatedAt = time.Now().UTC()
	e = p.DB.Collection("movies").FindOneAndUpdate(r.Context(), bson.M{"_id": id}, bson.M{"$set": bson.M{"title": x.Title, "description": x.Description, "category": x.Category, "language": x.Language, "durationMinutes": x.DurationMinutes, "rating": x.Rating, "year": x.Year, "tags": x.Tags, "posterUrl": x.PosterURL, "updatedAt": x.UpdatedAt}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&x)
	if e != nil {
		write(w, 404, nil)
		return
	}
	write(w, 200, map[string]any{"movie": x})
}
func (p *Platform) delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.current(r); !ok {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id, e := oid(r.PathValue("id"))
	if e != nil {
		write(w, 400, nil)
		return
	}
	res, e := p.DB.Collection("movies").DeleteOne(r.Context(), bson.M{"_id": id})
	if e != nil || res.DeletedCount == 0 {
		write(w, 404, nil)
		return
	}
	write(w, 200, map[string]string{"status": "deleted"})
}
func (p *Platform) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := bson.M{}
	if q != "" {
		filter["$text"] = bson.M{"$search": q}
	}
	if c := r.URL.Query().Get("category"); c != "" {
		filter["category"] = bson.M{"$regex": "^" + c + "$", "$options": "i"}
	}
	pageSize := 40
	if requested, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && requested > 0 && requested <= 100 {
		pageSize = requested
	}
	pageSkip := 0
	if requested, err := strconv.Atoi(r.URL.Query().Get("skip")); err == nil && requested > 0 {
		pageSkip = requested
	}
	cur, e := p.DB.Collection("movies").Find(r.Context(), filter, options.Find().SetLimit(int64(pageSize)).SetSkip(int64(pageSkip)))
	if e != nil {
		write(w, 500, map[string]string{"error": e.Error()})
		return
	}
	defer cur.Close(r.Context())
	var a []Movie
	_ = cur.All(r.Context(), &a)
	if a == nil {
		a = []Movie{}
	}
	write(w, 200, map[string]any{"movies": a, "count": len(a)})
}
func (p *Platform) event(w http.ResponseWriter, r *http.Request) {
	var x Event
	if body(r, &x) != nil {
		write(w, 400, nil)
		return
	}
	p.saveEvent(w, r, x)
}
func (p *Platform) eventForMovie(w http.ResponseWriter, r *http.Request) {
	id, e := oid(r.PathValue("id"))
	if e != nil {
		write(w, 400, nil)
		return
	}
	var x Event
	if body(r, &x) != nil {
		write(w, 400, nil)
		return
	}
	x.MovieID = id
	p.saveEvent(w, r, x)
}
func (p *Platform) saveEvent(w http.ResponseWriter, r *http.Request, x Event) {
	u, ok := p.current(r)
	if !ok {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	if x.Type != "view" && x.Type != "rating" && x.Type != "watchlist" && x.Type != "purchase" {
		write(w, 400, map[string]string{"error": "type must be view, rating, watchlist, or purchase"})
		return
	}
	if x.Type == "rating" && (x.Value < 1 || x.Value > 5) {
		write(w, 400, map[string]string{"error": "rating must be 1..5"})
		return
	}
	if x.MovieID.IsZero() {
		write(w, 400, nil)
		return
	}
	var movie Movie
	if err := p.DB.Collection("movies").FindOne(r.Context(), bson.M{"_id": x.MovieID}).Decode(&movie); err != nil {
		write(w, 404, map[string]string{"error": "movie not found"})
		return
	}
	x.UserID = u.ID
	x.MovieTitle = movie.Title
	x.CreatedAt = time.Now().UTC()
	res, e := p.DB.Collection("events").InsertOne(r.Context(), x)
	if e != nil {
		write(w, 500, nil)
		return
	}
	x.ID = res.InsertedID.(bson.ObjectID)
	write(w, 201, map[string]any{"interaction": x})
}
func (p *Platform) history(w http.ResponseWriter, r *http.Request) {
	u, ok := p.current(r)
	if !ok {
		write(w, 401, nil)
		return
	}
	cur, e := p.DB.Collection("events").Find(r.Context(), bson.M{"userId": u.ID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if e != nil {
		write(w, 500, nil)
		return
	}
	defer cur.Close(r.Context())
	var a []Event
	_ = cur.All(r.Context(), &a)
	if a == nil {
		a = []Event{}
	}
	write(w, 200, map[string]any{"history": a})
}
