package main

import (
	"analysis/data"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"time"
)

const (
	webPort  = "80"
	redisURL = "localhost:6379"
	mongoURL = "mongodb://mongo:27017"
)

var rclient *redis.Client
var mclient *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// connect to redis
	redisClient, err := connectToRedis()
	if err != nil {
		log.Println("error to connect mongo", err)
	}
	rclient = redisClient

	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Println("error to connect mongo", err)
	}
	mclient = mongoClient

	// create a context in order to disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// close connection
	defer func() {
		if err = mclient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	app := Config{
		Models: data.New(rclient, mclient),
	}

	// start web server
	log.Println("Starting service on port", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connectToRedis() (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		// TODO create env
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	if err := redisClient.Ping(context.TODO()).Err(); err != nil {
		log.Println("Error connecting:", err)
		return nil, err
	}

	log.Println("Connected to redis")

	return redisClient, nil
}

func connectToMongo() (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetAuth(options.Credential{
		// TODO create env
		Username: "admin",
		Password: "password",
	})

	// connect
	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("Error connecting:", err)
		return nil, err
	}

	log.Println("Connected to mongo")

	return c, nil
}
