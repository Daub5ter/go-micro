package data

import (
	"context"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var rclient *redis.Client
var mclient *mongo.Client

func New(redis *redis.Client, mongo *mongo.Client) Models {
	rclient = redis
	mclient = mongo

	return Models{
		ActionsUser:  ActionsUser{},
		AnalysisUser: AnalysisUser{},
	}
}

type Models struct {
	ActionsUser  ActionsUser
	AnalysisUser AnalysisUser
}

type ActionsUser struct {
	Email   string `json:"email"`
	Actions string `json:"actions"`
}

type AnalysisUser struct {
	ID              string    `bson:"_id,omitempty" json:"id,omitempty"`
	Email           string    `bson:"email" json:"email"`
	Actions         int       `bson:"actions" json:"actions"`
	PercentActivity float64   `bson:"percent_activity" json:"percent_activity"`
	CreatedAt       time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at" json:"updated_at"`
}

func (a *ActionsUser) Set(entry ActionsUser) error {
	err := rclient.Set(context.Background(), "actions "+entry.Email, entry.Actions, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (a *ActionsUser) Get(entry ActionsUser) (string, error) {
	val, err := rclient.Get(context.Background(), "actions "+entry.Email).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		} else {
			return "", err
		}
	}

	return val, nil
}

func (a *ActionsUser) GetAll() (map[string]string, error) {
	var cursor uint64
	var keyValues map[string]string
	for {
		keys, cursor, err := rclient.Scan(context.Background(), cursor, "actions *", 0).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			val, err := rclient.Get(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}

			keyValues[key] = val
		}

		if cursor == 0 {
			break
		}
	}

	return keyValues, nil
}

func (a *AnalysisUser) Insert(entry AnalysisUser) error {
	collection := mclient.Database("anal").Collection("anal")

	_, err := collection.InsertOne(context.TODO(), AnalysisUser{
		Email:           entry.Email,
		Actions:         entry.Actions,
		PercentActivity: entry.PercentActivity,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	if err != nil {
		log.Println("Error inserting into anal:", err)
		return err
	}

	return nil
}

func (a *AnalysisUser) All() ([]*AnalysisUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := mclient.Database("anal").Collection("anal")

	opts := options.Find()
	opts.SetSort(bson.D{{"created_at", -1}})

	cursor, err := collection.Find(context.TODO(), bson.D{}, opts)
	if err != nil {
		log.Println("Finding all docs error:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []*AnalysisUser

	for cursor.Next(ctx) {
		var item AnalysisUser

		err = cursor.Decode(&item)
		if err != nil {
			log.Println("Error decoding anal into slice:", err)
			return nil, err
		} else {
			logs = append(logs, &item)
		}
	}

	return logs, nil
}

func (a *AnalysisUser) GetOne(id string) (*AnalysisUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := mclient.Database("anal").Collection("anal")

	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var entry AnalysisUser
	err = collection.FindOne(ctx, bson.M{"_id": docID}).Decode(&entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (a *AnalysisUser) DropCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := mclient.Database("anal").Collection("anal")

	if err := collection.Drop(ctx); err != nil {
		return err
	}

	return nil
}

func (a *AnalysisUser) Update() (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := mclient.Database("anal").Collection("anal")

	docID, err := primitive.ObjectIDFromHex(a.ID)
	if err != nil {
		return nil, err
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": docID},
		bson.D{
			{"$set", bson.D{
				{"email", a.Email},
				{"actions", a.Actions},
				{"pecent_activity", a.PercentActivity},
				{"updated_at", time.Now()},
			}},
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}
