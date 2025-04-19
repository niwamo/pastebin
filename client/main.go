package main

import (
	"context"
	"time"
	"fmt"
	"os"
	"crypto/tls"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "pastebin/proto"
)

var (
	address string
)

func Connect(addr string) (pb.PasteBinClient, *grpc.ClientConn) {
	tlsConfig := &tls.Config {
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		fmt.Printf("ERROR: did not connect: %v\n", err)
		os.Exit(1)
	}
	client := pb.NewPasteBinClient(conn)
	return client, conn
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "pastebin-cli",
		Short: "A command-line gRPC client for pastebin",
	}

	rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "localhost:50051", "Server address")

	// `getBins` subcommand
	getBins := &cobra.Command{
		Use:   "getBins",
		Short: "Retrieves bins",
		Run: func(cmd *cobra.Command, args []string) {
			client, conn := Connect(address)
			defer conn.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r, err := client.GetBins(ctx, &pb.GetBinsRequest{})
			if err != nil {
				fmt.Printf("ERROR: could not get bins: %v\n", err)
				os.Exit(1)
			}
			for _, bin := range r.GetData() {
				fmt.Printf("time: %d\ttitle: %s\tcontent: %s\n", bin.Timestamp, bin.Title, bin.Content)
			}
		},
	}

	// `newBin` subcommand
	newBin := &cobra.Command{
		Use:   "newBin <title> <content>",
		Short: "Submits a new bin",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			title := args[0]
			content := args[1]
			client, conn := Connect(address)
			defer conn.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := client.NewBin(ctx, &pb.NewBinRequest{Title: title, Content: content})
			if err != nil {
				fmt.Printf("ERROR: could not add bin: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Success")
		},
	}

	// Register subcommands
	rootCmd.AddCommand(getBins, newBin)

	// Run CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
