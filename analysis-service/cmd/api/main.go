package main

import (
	"analysis/data"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

const (
	webPort  = "80"
	redisURL = "redis:6379"
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
		log.Println("error to connect redis", err)
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

	done := make(chan bool)
	go func() {
		for {
			go app.UpdateDB(done)
			<-done
		}
	}()

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connectToRedis() (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	var counts int64
	var backOff = 1 * time.Second

	// don`t continue until redis is ready
	for {
		rc := redis.NewClient(&redis.Options{
			Addr:     redisURL,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})

		err := redisClient.Ping(context.TODO()).Err()
		if err != nil {
			log.Println("Error connecting:", err)
			counts++
		} else {
			log.Println("Connected to redis")
			redisClient = rc
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("backing off")
		time.Sleep(backOff)
		continue
	}

	return redisClient, nil
}

func connectToMongo() (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetAuth(options.Credential{
		Username: os.Getenv("MONGO_USERNAME"),
		Password: os.Getenv("MONGO_PASSWORD"),
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
