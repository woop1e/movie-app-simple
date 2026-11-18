// Command import-movielens imports the GroupLens MovieLens latest-small data.
// It is idempotent and keeps the downloaded CSV files outside the repository.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"yozigo/internal/mongoplatform"
)

const batchSize = 500

var yearSuffix = regexp.MustCompile(`\s*\((\d{4})\)\s*$`)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(env("MONGODB_URI", "mongodb://localhost:27017")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	db := client.Database(env("MONGODB_DATABASE", "yozigo"))
	if err := (&mongoplatform.Platform{DB: db}).Indexes(ctx); err != nil {
		log.Fatal(err)
	}
	dir := env("MOVIELENS_DIR", "../work/datasets/movielens/ml-latest-small")
	movieIDs, movieCount, err := importMovies(ctx, db, dir+"/movies.csv")
	if err != nil {
		log.Fatal(err)
	}
	userCount, eventCount, err := importRatings(ctx, db, dir+"/ratings.csv", movieIDs)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Imported %d MovieLens titles, %d anonymized users, %d rating events into %s", movieCount, userCount, eventCount, db.Name())
}

func importMovies(ctx context.Context, db *mongo.Database, path string) (map[int64]bson.ObjectID, int, error) {
	known := make(map[string]bson.ObjectID)
	knownSource := make(map[int64]bson.ObjectID)
	cur, err := db.Collection("movies").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"title": 1, "year": 1, "source": 1, "sourceId": 1, "_id": 1}))
	if err != nil {
		return nil, 0, err
	}
	for cur.Next(ctx) {
		var row struct {
			ID       bson.ObjectID `bson:"_id"`
			Title    string        `bson:"title"`
			Year     int           `bson:"year"`
			Source   string        `bson:"source"`
			SourceID string        `bson:"sourceId"`
		}
		if err := cur.Decode(&row); err != nil {
			_ = cur.Close(ctx)
			return nil, 0, err
		}
		if row.Source == "MovieLens" {
			if sourceID, parseErr := strconv.ParseInt(row.SourceID, 10, 64); parseErr == nil {
				knownSource[sourceID] = row.ID
			}
		} else {
			known[movieKey(row.Title, row.Year)] = row.ID
		}
	}
	if err := cur.Err(); err != nil {
		_ = cur.Close(ctx)
		return nil, 0, err
	}
	_ = cur.Close(ctx)

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return nil, 0, err
	}
	movieIDs := make(map[int64]bson.ObjectID)
	models := make([]mongo.WriteModel, 0, batchSize)
	count := 0
	flush := func() error {
		if len(models) == 0 {
			return nil
		}
		_, err := db.Collection("movies").BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
		models = models[:0]
		return err
	}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, count, err
		}
		number, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, count, err
		}
		title, year := parseTitleYear(row[1])
		genres := splitGenres(row[2])
		id, matched := knownSource[number]
		if !matched {
			id, matched = known[movieKey(title, year)]
		}
		if !matched {
			id = numericObjectID(number)
		}
		movieIDs[number] = id
		if matched {
			// Preserve the Assignment #6 metadata and poster for titles already
			// present, while attaching the MovieLens source identifier.
			models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).SetUpdate(bson.M{"$set": bson.M{"source": "MovieLens", "sourceId": strconv.FormatInt(number, 10)}}))
		} else {
			category := "Other"
			if len(genres) > 0 {
				category = genres[0]
			}
			movie := mongoplatform.Movie{ID: id, Title: title, Description: "MovieLens catalog item; no synopsis supplied.", Category: category, Language: "Unknown", Rating: "NR", Year: year, Tags: genres, Source: "MovieLens", SourceID: strconv.FormatInt(number, 10), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
			doc, err := bson.Marshal(movie)
			if err != nil {
				return nil, count, err
			}
			var value bson.M
			if err := bson.Unmarshal(doc, &value); err != nil {
				return nil, count, err
			}
			models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).SetUpdate(bson.M{"$setOnInsert": value}).SetUpsert(true))
		}
		count++
		if len(models) >= batchSize {
			if err := flush(); err != nil {
				return nil, count, err
			}
		}
	}
	if err := flush(); err != nil {
		return nil, count, err
	}
	return movieIDs, count, nil
}

func importRatings(ctx context.Context, db *mongo.Database, path string, movieIDs map[int64]bson.ObjectID) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return 0, 0, err
	}
	users := make(map[int64]bson.ObjectID)
	userModels := make([]mongo.WriteModel, 0, 250)
	eventModels := make([]mongo.WriteModel, 0, batchSize)
	eventCount := 0
	flush := func(collection *mongo.Collection, models *[]mongo.WriteModel) error {
		if len(*models) == 0 {
			return nil
		}
		_, err := collection.BulkWrite(ctx, *models, options.BulkWrite().SetOrdered(false))
		*models = (*models)[:0]
		return err
	}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return len(users), eventCount, err
		}
		userNumber, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return len(users), eventCount, err
		}
		movieNumber, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			return len(users), eventCount, err
		}
		movieID, ok := movieIDs[movieNumber]
		if !ok {
			return len(users), eventCount, fmt.Errorf("ratings references unknown movie id %d", movieNumber)
		}
		userID, ok := users[userNumber]
		if !ok {
			userID = numericObjectID(userNumber)
			users[userNumber] = userID
			profile := bson.M{"_id": userID, "name": fmt.Sprintf("MovieLens Viewer %d", userNumber), "email": fmt.Sprintf("movielens-%d@example.invalid", userNumber), "role": "customer", "createdAt": time.Date(2018, 9, 1, 0, 0, 0, 0, time.UTC), "source": "MovieLens", "sourceId": strconv.FormatInt(userNumber, 10)}
			userModels = append(userModels, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": userID}).SetUpdate(bson.M{"$setOnInsert": profile}).SetUpsert(true))
		}
		rating, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return len(users), eventCount, err
		}
		seconds, err := strconv.ParseInt(row[3], 10, 64)
		if err != nil {
			return len(users), eventCount, err
		}
		eventID := hashedObjectID(fmt.Sprintf("ml-rating:%d:%d", userNumber, movieNumber))
		event := mongoplatform.Event{ID: eventID, UserID: userID, MovieID: movieID, Type: "rating", Value: rating, CreatedAt: time.Unix(seconds, 0).UTC(), Source: "MovieLens", SourceID: fmt.Sprintf("%d:%d", userNumber, movieNumber)}
		doc, err := bson.Marshal(event)
		if err != nil {
			return len(users), eventCount, err
		}
		var value bson.M
		if err := bson.Unmarshal(doc, &value); err != nil {
			return len(users), eventCount, err
		}
		eventModels = append(eventModels, mongo.NewReplaceOneModel().SetFilter(bson.M{"_id": eventID}).SetReplacement(value).SetUpsert(true))
		eventCount++
		if len(userModels) >= 250 {
			if err := flush(db.Collection("users"), &userModels); err != nil {
				return len(users), eventCount, err
			}
		}
		if len(eventModels) >= batchSize {
			if err := flush(db.Collection("events"), &eventModels); err != nil {
				return len(users), eventCount, err
			}
		}
	}
	if err := flush(db.Collection("users"), &userModels); err != nil {
		return len(users), eventCount, err
	}
	if err := flush(db.Collection("events"), &eventModels); err != nil {
		return len(users), eventCount, err
	}
	return len(users), eventCount, nil
}

func parseTitleYear(raw string) (string, int) {
	matches := yearSuffix.FindStringSubmatch(raw)
	if len(matches) < 2 {
		return canonicalArticle(strings.TrimSpace(raw)), 0
	}
	year, _ := strconv.Atoi(matches[1])
	return canonicalArticle(strings.TrimSpace(yearSuffix.ReplaceAllString(raw, ""))), year
}

func splitGenres(value string) []string {
	if value == "" || value == "(no genres listed)" {
		return nil
	}
	return strings.Split(value, "|")
}

func movieKey(title string, year int) string {
	return strings.ToLower(canonicalArticle(strings.TrimSpace(title))) + "|" + strconv.Itoa(year)
}

func canonicalArticle(title string) string {
	for _, article := range []string{"The", "An", "A"} {
		suffix := ", " + article
		if len(title) >= len(suffix) && strings.EqualFold(title[len(title)-len(suffix):], suffix) {
			return article + " " + strings.TrimSpace(title[:len(title)-len(suffix)])
		}
	}
	return title
}

func numericObjectID(number int64) bson.ObjectID {
	id, _ := bson.ObjectIDFromHex(fmt.Sprintf("%024x", uint64(number)))
	return id
}

func hashedObjectID(value string) bson.ObjectID {
	digest := sha256.Sum256([]byte(value))
	var id bson.ObjectID
	copy(id[:], digest[:12])
	return id
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
