package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/ZhongRuoyu/shorten/pkg/shortener"
	"golang.org/x/term"
)

func usage(exitCode int) {
	fmt.Fprintln(os.Stderr,
		"Usage: shortenpw <database> [create|check|update|delete] <username>")
	os.Exit(exitCode)
}

func readPassword(prompt string) (string, error) {
	if prompt != "" {
		fmt.Print(prompt)
	}

	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()

	password := string(passwordBytes)
	return password, nil
}

func promptPassword(confirm bool) string {
	password, err := readPassword("Enter password: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading password: ", err)
		os.Exit(1)
	}
	if !confirm {
		return password
	}

	confirmPassword, err := readPassword("Confirm password: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading password: ", err)
		os.Exit(1)
	}
	if password != confirmPassword {
		fmt.Fprintln(os.Stderr, "Passwords do not match")
		os.Exit(1)
	}
	return password
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 &&
		(args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		usage(0)
	}
	if len(args) != 3 {
		usage(1)
	}

	dbPath := args[0]
	action := args[1]
	username := args[2]

	db, err := shortener.NewDatabase(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening database: ", err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing database: ", err)
		os.Exit(1)
	}

	switch action {
	case "create":
		password := promptPassword(true)
		err = db.CreateUser(username, password)
		if err != nil {
			if err == shortener.ErrUsernameAlreadyInUse {
				fmt.Fprintln(os.Stderr, "User already exists")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error creating user: ", err)
			os.Exit(1)
		}
		fmt.Println("User created successfully")
	case "check":
		password := promptPassword(false)
		ok, err := db.CheckCredentials(username, password)
		if err != nil {
			if err == shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "User not found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error checking credentials: ", err)
			os.Exit(1)
		}
		if ok {
			fmt.Println("Credentials are correct")
		} else {
			fmt.Println("Credentials are incorrect")
		}
	case "update":
		password := promptPassword(true)
		err = db.UpdateCredentials(username, password)
		if err != nil {
			if err == shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "User not found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error updating credentials: ", err)
			os.Exit(1)
		}
		fmt.Println("Credentials updated successfully")
	case "delete":
		err = db.DeleteUser(username)
		if err != nil {
			if err == shortener.ErrNotFound {
				fmt.Fprintln(os.Stderr, "User not found")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error deleting user: ", err)
			os.Exit(1)
		}
		fmt.Println("User deleted successfully")
	default:
		usage(1)
	}
}
