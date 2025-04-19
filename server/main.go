package main

import (
	"fmt"
	"net"
	"net/http"
	"log"
	"context"
	"time"
	"os"
	"errors"
	"html/template"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "pastebin/proto"
)

var DatabaseName         string = "pastebin"
var ActiveCollectionName string = "active-bins"
var OldCollectionName    string = "old-bins"
var MaxBins              int64  = 10
var DB                   *mongo.Client
var URI                  string
var DISABLE_HTML_ESCAPE  bool
var ENABLE_GRPC          bool

type Bin struct {
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
	Title     string `bson:"title"     json:"title"`
	Content   string `bson:"content"   json:"content"`
}

// shared function
func getBins() ([]Bin, error) {
	collection := DB.Database(DatabaseName).Collection(ActiveCollectionName)
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	findOptions := options.Find().SetSort(bson.D{{"timestamp",1}})
	cursor, err := collection.Find(ctx, bson.D{}, findOptions)
	if err != nil {
		log.Printf("Error with bin retrieval: %s", err)
		return nil, err
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
	return results, nil
}

// HTTP wrapper
func getBinsHTTPHandler(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		log.Printf("Received illegal %s request to %s", request.Method, request.URL.Path)
		return
	}
	log.Printf("Received request for %s", request.URL.Path)
	
	results, err := getBins()
	if err != nil {
		log.Printf("Error getting bins: %s", err)
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
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

// gRPC wrapper
func (s *server) GetBins(_ context.Context, in *pb.GetBinsRequest) (*pb.GetBinsReply, error) {
	log.Printf("Received request to gRPC GetBins")
	results, err := getBins()
	if err != nil {
		log.Printf("Error getting bins: %s", err)
		return nil, err
	}
	var out []*pb.Bin
	for _, item := range results {
		tmp := pb.Bin{
			Timestamp: item.Timestamp,
			Title: item.Title,
			Content: item.Content,
		}
		out = append(out, &tmp)
	}
	return &pb.GetBinsReply{Data: out}, nil
}

// shared function
func newBin(title string, content string) (int32, error) {
	if len(title) > 20 || len(content) > 256 {
		return -1, errors.New("Request too large")
	}
	bin := Bin{
		Timestamp: time.Now().Unix(),
		Title:     title,
		Content:   content,
	}
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	activeCollection := DB.Database(DatabaseName).Collection(ActiveCollectionName)
	opts := options.Count().SetHint("_id_")
	count, err := activeCollection.CountDocuments(context.TODO(), bson.D{}, opts)
	if err != nil {
		log.Print("Could not count documents")
		return -1, errors.New("Database error")
	}
	if count >= MaxBins {
		findOptions := options.FindOneAndReplace().SetSort(bson.D{{"timestamp", 1}})
		var oldBin Bin
		err = activeCollection.FindOneAndReplace(ctx, bson.D{}, bin, findOptions).Decode(&oldBin)
		if err != nil {
			log.Printf("Error adding bin: %s", err)
			return -1, errors.New("Error adding bin")
		}
		oldCollection := DB.Database(DatabaseName).Collection(OldCollectionName)
		_, err = oldCollection.InsertOne(ctx, oldBin)
		if err != nil {
			log.Printf("Error adding bin to old collection: %s", err)
		}
		return 0, nil
	}
	// if fewer than 25
	_, err = activeCollection.InsertOne(ctx, bin)
	if err != nil {
		log.Printf("Error adding bin: %s", err)
		return -1, errors.New("Error adding bin")
	}
	return 0, nil
}

// HTTP wrapper
func newBinHTTPHandler(response http.ResponseWriter, request *http.Request) {
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
	if ! DISABLE_HTML_ESCAPE {
		content = template.HTMLEscapeString(content)
	}
	
	_, err = newBin(request.FormValue("title"), content)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(response, "Success")
}

// gRPC wrapper
func (s *server) NewBin(_ context.Context, in *pb.NewBinRequest) (*pb.NewBinResponse, error) {
	log.Printf("Received request to gRPC NewBin")
	_, err := newBin(in.Title, in.Content)
	if err != nil {
		return nil, err
	}
	return &pb.NewBinResponse{Status: 200}, nil
}

// default HTTP handler
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

// web server to be fun as go routine
func ServeWeb() {
	http.HandleFunc("/", getStatic)
	http.HandleFunc("/api/v1.0/getBins", getBinsHTTPHandler)
	http.HandleFunc("/api/v1.0/newBin", newBinHTTPHandler)
	err := http.ListenAndServeTLS(":443", "/cert.crt", "/cert.key", nil)
	if err != nil {
		log.Print(err)
	}
	log.Print("Exiting.")
}

type server struct {
	pb.UnimplementedPasteBinServer
}

// gRPC server to be fun as go routine
func ServeGRPC() {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 50051))
	if err != nil {
		log.Printf("gRPC failed to listen. Exiting. Error message: %v", err)
		return
	}
	creds, err := credentials.NewServerTLSFromFile("/cert.crt", "/cert.key")
	if err != nil {
		log.Printf("gRPC failed to load TLS certs. Exiting. Error: %v", err)
		return
	}
	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterPasteBinServer(s, &server{})
	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	log.Print("Starting server(s)...")

	URI = os.Getenv("DB_CONN_STRING")
	log.Printf("DB_CONN_STRING: %s", URI)
	DISABLE_HTML_ESCAPE = os.Getenv("DISABLE_HTML_ESCAPE") == "1"
	log.Printf("DISABLE_HTML_ESCAPE: %s", DISABLE_HTML_ESCAPE)
	ENABLE_GRPC = os.Getenv("ENABLE_GRPC") == "1"
	log.Printf("ENABLE_GRPC: %s", ENABLE_GRPC)

	// connect to database
	client, err := mongo.NewClient(options.Client().ApplyURI(URI))
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
	DB = client
	
	go ServeWeb()
	if ENABLE_GRPC { go ServeGRPC() }

	for {
		time.Sleep(time.Second)
	}
}