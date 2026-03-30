package main

import (
	"fmt"
	"os"

	"github.com/ZhongRuoyu/shorten/pkg/shortener"
)

const (
	usageText = `Usage: shortenkey <database> <action> [args...]
Actions:
  create-user <username>
  list-users
  delete-user <username>
  create-key  <username>
  check-key   <key|key-hash>
  list-keys   <username>
  delete-key  <key|key-hash>
`
)

var (
	actionArgc = map[string]int{
		"create-user": 1,
		"list-users":  0,
		"delete-user": 1,
		"create-key":  1,
		"check-key":   1,
		"list-keys":   1,
		"delete-key":  1,
	}
)

func usage(exitCode int) {
	fmt.Fprint(os.Stderr, usageText)
	os.Exit(exitCode)
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 &&
		(args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		usage(0)
	}
	if len(args) < 2 {
		usage(1)
	}

	dbPath := args[0]
	action := args[1]
	arg := ""

	expectedArgc, ok := actionArgc[action]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown action:", action)
		usage(1)
	}
	if len(args)-2 != expectedArgc {
		fmt.Fprintf(os.Stderr, "Action %s expects %d argument(s)\n",
			action, expectedArgc)
		usage(1)
	}
	if expectedArgc > 0 {
		arg = args[2]
	}

	db, err := shortener.NewDatabase(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening database:", err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing database:", err)
		os.Exit(1)
	}

	switch action {
	case "create-user":
		err = db.CreateUser(arg)
		if err != nil {
			if err == shortener.ErrUsernameAlreadyInUse {
				fmt.Fprintln(os.Stderr, "User already exists")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error creating user:", err)
			os.Exit(1)
		}
		fmt.Println("User created successfully")
	case "list-users":
		users, err := db.ListUsers()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error listing users:", err)
			os.Exit(1)
		}
		for _, user := range users {
			fmt.Println(user)
		}
	case "delete-user":
		err = db.DeleteUser(arg)
		if err != nil {
			if err == shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "User not found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error deleting user:", err)
			os.Exit(1)
		}
		fmt.Println("User deleted successfully")
	case "create-key":
		key, err := db.CreateApiKey(arg)
		if err != nil {
			if err == shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "User not found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error creating API key:", err)
			os.Exit(1)
		}
		fmt.Println(key)
	case "check-key":
		username, err := db.CheckApiKey(arg)
		if err != nil {
			if err != shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "Error checking API key:", err)
				os.Exit(1)
			}
			username, err = db.CheckApiKeyByHash(arg)
			if err != nil {
				if err == shortener.ErrNotFound {
					fmt.Fprintln(os.Stderr, "API key not valid")
					os.Exit(1)
				}
				fmt.Fprintln(os.Stderr, "Error checking API key:", err)
				os.Exit(1)
			}
		}
		fmt.Printf("Valid (user: %s)\n", username)
	case "list-keys":
		keys, err := db.ListApiKeys(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error listing API keys:", err)
			os.Exit(1)
		}
		for _, key := range keys {
			fmt.Println(key)
		}
	case "delete-key":
		err = db.DeleteApiKey(arg)
		if err != nil {
			if err != shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "Error deleting API key:", err)
				os.Exit(1)
			}
			err = db.DeleteApiKeyByHash(arg)
			if err != nil {
				if err == shortener.ErrNotFound {
					fmt.Fprintln(os.Stderr, "API key not found")
					os.Exit(1)
				}
				fmt.Fprintln(os.Stderr, "Error deleting API key:", err)
				os.Exit(1)
			}
		}
		fmt.Println("API key deleted successfully")
	}
}
