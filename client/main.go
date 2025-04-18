package main

import (
	"flag"
	"log"
	"context"
	"time"
	"fmt"
	"slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "pastebin/proto"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

func main() {
	flag.Parse()

	cmd := flag.Arg(0)
	if !slices.Contains([]string{"getBins", "newBin"}, cmd) {
		log.Fatalf("ERROR: UNKNOWN COMMAND")
		return
	}

	// Set up a connection to the server.
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPasteBinClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	if (cmd == "getBins") {
		r, err := client.GetBins(ctx, &pb.GetBinsRequest{})
		if err != nil {
			log.Fatalf("could not get bins: %v", err)
		}
		for _, bin := range r.GetData() {
			fmt.Printf("time: %d\ttitle: %s\tcontent: %s\n", bin.Timestamp, bin.Title, bin.Content)
		}
	} 
	if (cmd == "newBin") {
		if flag.NArg() < 3 {
			log.Fatalf("ERROR: NOT ENOUGH ARGS")
			return
		}
		_, err := client.NewBin(ctx, &pb.NewBinRequest{Title: flag.Arg(1), Content: flag.Arg(2)})
		if err != nil {
			log.Fatalf("could not add bin: %v", err)
		}
		fmt.Println("Success")
	} 

	return
}