package main

import (
	"WebApp/store"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
)

var connections int = 0

type taskID string

func GetHandler(res http.ResponseWriter, req *http.Request) {
	// traceID := req.Context().Value(taskID("taskID")).(string)
	// fmt.Printf("TraceID: %s | landing page\n", traceID)

	connections += 1
	fmt.Fprintf(res, "request- %d", connections)
}

func ListItemsHandler(res http.ResponseWriter, req *http.Request) {
	traceID := req.Context().Value(taskID("taskID")).(string)
	fmt.Printf("TraceID: %s | Adding item\n", traceID)
	todoItems := Store.ListItems()

	slog.Info("Item listed successfully", slog.String("TraceID", traceID))

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(todoItems)
}

func AddItemHandler(res http.ResponseWriter, req *http.Request) {
	traceID := req.Context().Value(taskID("taskID")).(string)
	fmt.Printf("TraceID: %s | Adding item\n", traceID)
	err := req.ParseForm()
	if err != nil {
		http.Error(res, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	item := req.FormValue("item")
	status := req.FormValue("status")
	if item == "" || status == "" {
		http.Error(res, "Invalid input", http.StatusBadRequest)
		return
	}
	insertToDo := Store.InsertItem(item, status)
	// if insertToDo  {
	// 	http.Error(res, "Failed to insert data", http.StatusInternalServerError)
	// 	slog.Info("Item add failed", slog.String("TraceID", traceID), slog.String("item", item))
	// }

	slog.Info("Item added successfully", slog.String("TraceID", traceID), slog.String("item", item))

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(insertToDo)
}

func UpdateItemHandler(res http.ResponseWriter, req *http.Request) {
	traceID := req.Context().Value(taskID("taskID")).(string)
	fmt.Printf("TraceID: %s | Editing item\n", traceID)
	item := req.FormValue("item")
	if len(item) == 0 {
		http.Error(res, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	err := Store.UpdateItem(item)
	if err != nil {
		http.Error(res, "Failed to update item", http.StatusInternalServerError)
		slog.Info("Item update failed", slog.String("TraceID", traceID), slog.String("item", item))
	} else {
		slog.Info("Item updated successfully", slog.String("TraceID", traceID), slog.String("item", item))
	}
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(store.ToDoItem{Item: item})
}

func deleteItemHandler(res http.ResponseWriter, req *http.Request) {
	traceID := req.Context().Value(taskID("taskID")).(string)
	fmt.Printf("\nTraceID: %s | Deleting item\n", traceID)
	item := req.FormValue("item")
	if len(item) == 0 {
		http.Error(res, "Failed to parse form data", http.StatusBadRequest)
		return
	}
	err := Store.DeleteItem(item)
	if err != nil {
		http.Error(res, "Failed to delete item", http.StatusInternalServerError)
		slog.Info(err.Error(), slog.String("TraceID", traceID), slog.String("item", item))
	} else {
		slog.Info("Item deleted successfully", slog.String("TraceID", traceID), slog.String("item", item))
	}
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode("item deleted")
}

func TheLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("%s %s\n", req.Method, req.URL.Path)
		taskContext := context.WithValue(req.Context(), taskID("taskID"), uuid.New().String())
		next.ServeHTTP(rw, req.WithContext(taskContext))
	})
}
func init() {
	// Set up slog to log to the standard output
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
func handleCliCommands() {
	fmt.Println(os.Args)
	args := os.Args[2:]

	insert := flag.NewFlagSet("insert", flag.ExitOnError)
	item := insert.String("item", "Default Item", "Todo item to add")
	status := insert.String("status", "pending", "Status of the Todo item")

	delete := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteItem := delete.String("item", "", "item to delete")

	update := flag.NewFlagSet("update", flag.ExitOnError)
	updateItem := update.String("item", "", "Todo item to add")
	updateStatus := update.String("status", "", "Status of the Todo item")

	switch args[0] {
	case "insert":
		insert.Parse(args[1:])
		Store.InsertItem(*item, *status)
	case "update":
		update.Parse(args[1:])
		Store.UpdateItem(*updateItem)
		fmt.Printf("%s is updated to %s", *updateItem, *updateStatus)
	case "delete":
		delete.Parse(args[1:])
		Store.DeleteItem(*deleteItem)
		fmt.Printf("%s is deleted", *deleteItem)
	case "list":
		fmt.Print("list")
		Store.ListItems()
	default:
		fmt.Print("Not a valid option, Available options are insert, delete, update, list")
	}
}

var Store store.ToDoStore

func main() {
	var err error
	var mode string

	mux := http.NewServeMux()
	// handler := http.HandlerFunc(GetHandler)
	mux.HandleFunc("/list", ListItemsHandler)
	mux.HandleFunc("/add", AddItemHandler)
	mux.HandleFunc("/edit", UpdateItemHandler)
	mux.HandleFunc("/delete", deleteItemHandler)
	mux.HandleFunc("/", GetHandler)
	ctx := context.Background()
	logger := TheLogger(mux)
	// Choose between cli and api modes
	flag.StringVar(&mode, "mode", "cli", "Run mode: 'cli' or 'api'")
	//Choose the Database
	// storeType := flag.String("store", "memory", "choose memory or json")
	flag.Parse()
	storeType := "json"
	fmt.Println("Store type...----->>>>", storeType)
	if storeType == "memory" {
		fmt.Println("Using MEMORY store ")
		Store = store.NewInMemoryStore(ctx)
	} else {
		fmt.Println("Using JSON store ")
		Store = store.NewJsonMemoryStore(ctx)
	}

	if mode == "cli" {
		handleCliCommands()
	} else {
		fmt.Println("starting server on port 8080.....")

		err = http.ListenAndServe(":8080", logger)
		if err != nil {
			fmt.Printf("server- %d is shutting down", connections)
		}

	}

}
