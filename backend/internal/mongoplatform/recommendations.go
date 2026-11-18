package mongoplatform

import (
	"math"
	"net/http"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const recommendationLimit = 20
const neighborhoodLimit = 20

type recommendationInteraction struct {
	UserID  bson.ObjectID `bson:"userId"`
	MovieID bson.ObjectID `bson:"movieId"`
	Type    string        `bson:"type"`
	Value   float64       `bson:"value"`
}

type interactionState struct {
	preference float64
	hasRating  bool
}

type scoredMovie struct {
	movie Movie
	score float64
}

func interactionPreference(kind string, value float64) float64 {
	switch kind {
	case "rating":
		// Ratings are centered at neutral (3/5) and scaled to [-1, 1].
		return (value - 3) / 2
	case "watchlist":
		return 0.8
	case "purchase":
		return 1
	case "view":
		return 0.2
	default:
		return 0
	}
}

func (p *Platform) recommend(w http.ResponseWriter, r *http.Request) {
	user, ok := p.current(r)
	if !ok {
		write(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	events, err := p.DB.Collection("events").Find(r.Context(), bson.M{}, options.Find().
		SetProjection(bson.M{"_id": 0, "userId": 1, "movieId": 1, "type": 1, "value": 1, "createdAt": 1}).
		SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not load interaction history"})
		return
	}
	defer events.Close(r.Context())

	// Collapse repeated actions to one preference per user/movie. The cursor is
	// newest-first so the latest explicit rating wins, while implicit actions
	// contribute their strongest signal unless an explicit rating exists.
	preferences := make(map[bson.ObjectID]map[bson.ObjectID]interactionState)
	seen := make(map[bson.ObjectID]struct{})
	for events.Next(r.Context()) {
		var event recommendationInteraction
		if err := events.Decode(&event); err != nil {
			write(w, http.StatusInternalServerError, map[string]string{"error": "could not read interaction history"})
			return
		}
		byMovie := preferences[event.UserID]
		if byMovie == nil {
			byMovie = make(map[bson.ObjectID]interactionState)
			preferences[event.UserID] = byMovie
		}
		state := byMovie[event.MovieID]
		if event.UserID == user.ID {
			seen[event.MovieID] = struct{}{}
		}
		if event.Type == "rating" {
			if !state.hasRating {
				state.preference = interactionPreference(event.Type, event.Value)
				state.hasRating = true
			}
		} else if !state.hasRating {
			if signal := interactionPreference(event.Type, event.Value); signal > state.preference {
				state.preference = signal
			}
		}
		byMovie[event.MovieID] = state
	}
	if err := events.Err(); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not read interaction history"})
		return
	}

	// User-based collaborative filtering: cosine similarity over positive
	// weighted interactions, with a small overlap shrinkage to avoid treating a
	// single shared view as a fully reliable match.
	target := positiveVector(preferences[user.ID])
	neighbors := make([]struct {
		id    bson.ObjectID
		score float64
	}, 0, len(preferences))
	for otherID, items := range preferences {
		if otherID == user.ID {
			continue
		}
		vector := positiveVector(items)
		similarity, overlap := cosineSimilarity(target, vector)
		if similarity > 0 {
			neighbors = append(neighbors, struct {
				id    bson.ObjectID
				score float64
			}{otherID, similarity * float64(overlap) / float64(overlap+2)})
		}
	}
	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].score > neighbors[j].score })
	if len(neighbors) > neighborhoodLimit {
		neighbors = neighbors[:neighborhoodLimit]
	}

	numerator := make(map[bson.ObjectID]float64)
	denominator := make(map[bson.ObjectID]float64)
	for _, neighbor := range neighbors {
		for movieID, state := range preferences[neighbor.id] {
			if _, alreadySeen := seen[movieID]; alreadySeen || state.preference == 0 {
				continue
			}
			numerator[movieID] += neighbor.score * state.preference
			denominator[movieID] += neighbor.score
		}
	}

	candidateIDs := make([]bson.ObjectID, 0, len(numerator))
	for movieID, weightedSignal := range numerator {
		if weightedSignal > 0 && denominator[movieID] > 0 {
			candidateIDs = append(candidateIDs, movieID)
		}
	}
	filter := bson.M{"_id": bson.M{"$in": candidateIDs}}
	cur, err := p.DB.Collection("movies").Find(r.Context(), filter)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not load movie catalog"})
		return
	}
	defer cur.Close(r.Context())
	var catalog []Movie
	if err := cur.All(r.Context(), &catalog); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not read movie catalog"})
		return
	}

	ranked := make([]scoredMovie, 0)
	for _, movie := range catalog {
		if weight := denominator[movie.ID]; weight > 0 {
			if score := numerator[movie.ID] / weight; score > 0 {
				ranked = append(ranked, scoredMovie{movie: movie, score: score})
			}
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].score-ranked[j].score) < 1e-9 {
			return ranked[i].movie.Title < ranked[j].movie.Title
		}
		return ranked[i].score > ranked[j].score
	})

	algorithm := "User-based collaborative filtering (cosine similarity over weighted interactions)"
	if len(ranked) == 0 {
		// Cold-start path: retain useful recommendations before the user has
		// enough positive interactions to find similar viewers.
		algorithm = "Cold-start fallback (profile genres and newest catalog titles)"
		fallbackFilter := bson.M{}
		if len(seen) > 0 {
			seenIDs := make([]bson.ObjectID, 0, len(seen))
			for id := range seen {
				seenIDs = append(seenIDs, id)
			}
			fallbackFilter["_id"] = bson.M{"$nin": seenIDs}
		}
		if len(user.FavoriteGenres) > 0 {
			fallbackFilter["category"] = bson.M{"$in": user.FavoriteGenres}
		}
		fallback, err := p.DB.Collection("movies").Find(r.Context(), fallbackFilter,
			options.Find().SetSort(bson.D{{Key: "year", Value: -1}, {Key: "title", Value: 1}}).SetLimit(recommendationLimit))
		if err != nil {
			write(w, http.StatusInternalServerError, map[string]string{"error": "could not load fallback recommendations"})
			return
		}
		var movies []Movie
		if err := fallback.All(r.Context(), &movies); err != nil {
			_ = fallback.Close(r.Context())
			write(w, http.StatusInternalServerError, map[string]string{"error": "could not read fallback recommendations"})
			return
		}
		_ = fallback.Close(r.Context())
		if movies == nil {
			movies = []Movie{}
		}
		write(w, http.StatusOK, map[string]any{"movies": movies, "algorithm": algorithm, "similarUsers": 0})
		return
	}

	limit := len(ranked)
	if limit > recommendationLimit {
		limit = recommendationLimit
	}
	movies := make([]Movie, 0, limit)
	for _, item := range ranked[:limit] {
		movies = append(movies, item.movie)
	}
	write(w, http.StatusOK, map[string]any{"movies": movies, "algorithm": algorithm, "similarUsers": len(neighbors)})
}

func positiveVector(items map[bson.ObjectID]interactionState) map[bson.ObjectID]float64 {
	vector := make(map[bson.ObjectID]float64)
	for movieID, state := range items {
		if state.preference > 0 {
			vector[movieID] = state.preference
		}
	}
	return vector
}

func cosineSimilarity(a, b map[bson.ObjectID]float64) (float64, int) {
	var dot, normA, normB float64
	for _, value := range a {
		normA += value * value
	}
	for _, value := range b {
		normB += value * value
	}
	if normA == 0 || normB == 0 {
		return 0, 0
	}
	overlap := 0
	for movieID, value := range a {
		if other, ok := b[movieID]; ok {
			dot += value * other
			overlap++
		}
	}
	if overlap == 0 {
		return 0, 0
	}
	return dot / math.Sqrt(normA*normB), overlap
}
