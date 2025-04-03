package main

import (
	"fmt"
	"net/http"
	"log"
	"context"
	"time"
	"os"
	"html/template"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DatabaseName         string = "pastebin"
var ActiveCollectionName string = "active-bins"
var OldCollectionName    string = "old-bins"
var MaxBins              int64  = 10

type Bin struct {
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
	Title     string `bson:"title"     json:"title"`
	Content   string `bson:"content"   json:"content"`
}

func getGetBinsHandler(client *mongo.Client) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			log.Printf("Received illegal %s request to %s", request.Method, request.URL.Path)
			return
		}
		log.Printf("Received request for %s", request.URL.Path)
		
		collection := client.Database(DatabaseName).Collection(ActiveCollectionName)

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		findOptions := options.Find().SetSort(bson.D{{"timestamp",1}})
		cursor, err := collection.Find(ctx, bson.D{}, findOptions)
		if err != nil {
			log.Printf("Error with bin retrieval: %s", err)
			http.Error(response, "Error", http.StatusInternalServerError)
			return
		} 
		defer cursor.Close(ctx)
		var results []Bin
		for cursor.Next(ctx) {
			var bin Bin
			if err := cursor.Decode(&bin); err != nil {
				log.Printf("Error parsing MongoDB response: %s", err)
			}
			results = append(results, bin)
		}
		jsonData, err := json.Marshal(results)
		if err != nil {
			log.Printf("Error with JSON encoding: %s", err)
			http.Error(response, "Error", http.StatusInternalServerError)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		fmt.Fprint(response, string(jsonData))
		return
	}
}

func getNewBinHandler(client *mongo.Client, disableHTMLEscape bool) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			log.Printf("Received illegal %s request to %s", request.Method, request.URL.Path)
			return
		}
		log.Printf("Received request to %s", request.URL.Path)

		if request.ContentLength > 512 {
			log.Print("Request size exceeded")
			http.Error(response, "Request was too large", http.StatusBadRequest)
			return
		}

		err := request.ParseForm()
		if err != nil {
			http.Error(response, "Error parsing form", http.StatusBadRequest)
			return
		}
		
		content := request.FormValue("content")
		if ! disableHTMLEscape {
			content = template.HTMLEscapeString(content)
		}
		bin := Bin{
			Timestamp: time.Now().Unix(),
			Title:     request.FormValue("title"),
			Content:   content,
		}

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		activeCollection := client.Database(DatabaseName).Collection(ActiveCollectionName)

		opts := options.Count().SetHint("_id_")
		count, err := activeCollection.CountDocuments(context.TODO(), bson.D{}, opts)
		if err != nil {
			log.Print("Could not count documents")
			http.Error(response, "Error", http.StatusInternalServerError)
			return
		}
		if count >= MaxBins {
			findOptions := options.FindOneAndReplace().SetSort(bson.D{{"timestamp", 1}})
			var oldBin Bin
			err = activeCollection.FindOneAndReplace(ctx, bson.D{}, bin, findOptions).Decode(&oldBin)
			if err != nil {
				log.Printf("Error adding bin: %s", err)
				http.Error(response, "Error adding bin", http.StatusInternalServerError)
				return
			}
			oldCollection := client.Database(DatabaseName).Collection(OldCollectionName)
			_, err = oldCollection.InsertOne(ctx, oldBin)
			if err != nil {
				log.Printf("Error adding bin to old collection: %s", err)
			}
			return
		}
		// if fewer than 25
		_, err = activeCollection.InsertOne(ctx, bin)
		if err != nil {
			log.Printf("Error adding bin: %s", err)
			http.Error(response, "Error adding bin", http.StatusBadRequest)
			return
		}
		fmt.Fprint(response, "Success")
	}
}

func getStatic(response http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	if request.Method != "GET" {
		log.Printf("Received illegal %s request to %s", request.Method, path)
		return
	}
	log.Printf("Received request for %s", path)
	if (path[len(path)-1] == byte('/')) {
		path += "index.html"
	}
	filepath := fmt.Sprintf("/var/www/html%s", path)
	if _, err := os.Stat(filepath); err == nil {
		log.Printf("Returning file: %s", filepath)
		http.ServeFile(response, request, filepath)
		return
	} else {
		extPath := fmt.Sprintf("%s.html", filepath)
		if _, err := os.Stat(extPath); err == nil {
			log.Printf("Returning file: %s", extPath)
			http.ServeFile(response, request, extPath)
			return
		}
		log.Printf("Path not found: %s", filepath)
		http.Error(response, "Path not found", http.StatusNotFound)
		return
	}
}

func main() {
	log.Print("Starting server...")

	// retrieve configuration variables from environment
	uri := os.Getenv("DB_CONN_STRING")
	log.Printf("DB_CONN_STRING: %s", uri)
	disableHTMLEscape := os.Getenv("DISABLE_HTML_ESCAPE") == "1"
	log.Printf("DISABLE_HTML_ESCAPE: %s", disableHTMLEscape)

	// connect to database
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Printf("Error creating mongoDB client: %s", err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Printf("Error connecting to mongoDB: %s", err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("Error pinging database: %s", err)
	} else {
		log.Print("Connected to database")
	}

	// default handler
	http.HandleFunc("/", getStatic)

	// API handlers
	getBinsHandler := getGetBinsHandler(client)
	http.HandleFunc("/api/v1.0/getBins", getBinsHandler)
	newBinHandler := getNewBinHandler(client, disableHTMLEscape)
	http.HandleFunc("/api/v1.0/newBin", newBinHandler)

	// start server
	err = http.ListenAndServeTLS(":443", "/cert.crt", "/cert.key", nil)
	if err != nil {
		log.Print(err)
	}
	log.Print("Exiting.")
}