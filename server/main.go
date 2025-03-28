package main

import (
	"fmt"
	"net/http"
	"log"
	"context"
	"time"
	"strings"
	"os"
	"html/template"
	"regexp"
	"strconv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Bin struct {
	Title string `bson:"title"`
	Content string `bson:"content"`
}

func getRootHandler(client *mongo.Client, tmpl *template.Template) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			log.Printf("Received illegal %s request to /", request.Method)
			return
		}
		log.Print("Received request for /")
		
		collection := client.Database("aws-demo").Collection("bins")

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		cursor, err := collection.Find(ctx, bson.D{})
		var data map[string]template.HTML
		if err != nil {
			log.Print(err)
			data = map[string]template.HTML{
				"Bins": template.HTML("<tr><td>Error</td><td>Retrieving bins</td></tr>"),
			}
		} else {
			defer cursor.Close(ctx)
			var results []string
			for cursor.Next(ctx) {
				var bin Bin
				if err := cursor.Decode(&bin); err != nil {
					log.Fatal(err)
				}
				row := fmt.Sprintf("<tr><td>%s</td><td>%s</td>", bin.Title, bin.Content)
				results = append(results, row)
			}
			result := strings.Join(results, "\n")
			data = map[string]template.HTML{
				"Bins": template.HTML(result),
			}
		}
		
		err = tmpl.Execute(response, data)

		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func getSubmitHandler(client *mongo.Client) http.HandlerFunc {
	disableHTMLEscape := os.Getenv("DISABLE_HTML_ESCAPE")
	log.Printf("DISABLE_HTML_ESCAPE: %s", disableHTMLEscape)
	
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			log.Printf("Received illegal %s request to /submit", request.Method)
			return
		}
		log.Print("Received request to /submit")

		if request.ContentLength > 512 {
			log.Print("Request size exceeded")
			http.Error(response, "Bin size exceeded", http.StatusBadRequest)
			return
		}

		err := request.ParseForm()
		if err != nil {
			http.Error(response, "Error parsing form", http.StatusBadRequest)
			return
		}
		
		content := request.FormValue("content")
		if disableHTMLEscape != "1"	{
			content = template.HTMLEscapeString(content)
		}
		bin := Bin{
			Title: request.FormValue("title"),
			Content: content,
		}

		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		collection := client.Database("aws-demo").Collection("bins")

		opts := options.Count().SetHint("_id_")
		count, err := collection.CountDocuments(context.TODO(), bson.D{}, opts)
		if err != nil {
			log.Print("Could not count documents")
			http.Error(response, "Error", http.StatusInternalServerError)
			return
		}
		if count > 25 {
			http.Error(response, "Too many bins already", http.StatusBadRequest)
			return
		}

		_, err = collection.InsertOne(ctx, bin)
		if err != nil {
			log.Print("Error adding bin")
			http.Error(response, "Error adding bin", http.StatusBadRequest)
			return
		}

		fmt.Fprint(response, "Success")
	}
}

func getClearHandler(client *mongo.Client) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			log.Printf("Received illegal %s request to /submit", request.Method)
			return
		}
		log.Print("Received request to /submit")

		dbName := "aws-demo"
		ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
		collections, err := client.Database(dbName).ListCollectionNames(ctx, bson.D{})
		if err != nil {
			log.Printf("Error clearing %s", err)
			http.Error(response, "Error clearing", http.StatusInternalServerError)
			return
		}

		re := regexp.MustCompile(`^bkp-(\d+)$`)
		maxBackup := 0

		for _, name := range collections {
			matches := re.FindStringSubmatch(name)
			if len(matches) == 2 {
				num, err := strconv.Atoi(matches[1])
				if err == nil && num > maxBackup {
					maxBackup = num
				}
			}
		}

		oldName := "bins"
		newName := fmt.Sprintf("bkp-%d", maxBackup+1)
		cmd := bson.D{{"renameCollection", dbName + "." + oldName}, {"to", dbName + "." + newName}, {"dropTarget", false}}
		
		err = client.Database("admin").RunCommand(ctx, cmd).Err()
		if err != nil {
			log.Printf("Error clearing %s", err)
			http.Error(response, "Error clearing", http.StatusInternalServerError)
			return
		}

		log.Printf("Collection '%s' renamed to '%s' successfully.\n", oldName, newName)

		fmt.Fprint(response, "Success")
	}
}

func dbErrorServer() {
	log.Print("Running db error server")
	getRoot := func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "Database error", http.StatusInternalServerError)
	}
	http.HandleFunc("/", getRoot)
	err := http.ListenAndServeTLS(":443", "/opt/cert.crt", "/opt/cert.key", nil)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	log.Print("Starting server...")

	uri := os.Getenv("DB_CONN_STRING")
	log.Printf("DB_CONN_STRING: %s", uri)

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Print(err)
		dbErrorServer()
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Print(err)
		dbErrorServer()
		return
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("Error pinging database: %s", err)
	} else {
		log.Print("Connected to database")
	}

	rootTemplate, err := template.ParseFiles("/opt/index.html")
	if err != nil {
		log.Fatal(err)
	}

	getRoot := getRootHandler(client, rootTemplate)
	http.HandleFunc("/", getRoot)

	getSubmit := getSubmitHandler(client)
	http.HandleFunc("/submit", getSubmit)

	getClear := getClearHandler(client)
	http.HandleFunc("/clear", getClear)

	err = http.ListenAndServeTLS(":443", "/opt/cert.crt", "/opt/cert.key", nil)
	if err != nil {
		log.Print(err)
	}

	log.Print("Exiting.")
}