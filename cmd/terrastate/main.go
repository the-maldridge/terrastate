package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/the-maldridge/TerraState/internal/auth"
	_ "github.com/the-maldridge/TerraState/internal/auth/all"

	"github.com/hashicorp/terraform/state"
)

var (
	authService auth.Service

	statePath = flag.String("state_file", "./state.dat", "Location for the state file")

	addr = flag.String("bind_addr", "localhost", "Address to bind to")
	port = flag.Int("bind_port", 8081, "Port to bind to")
)

func manageState(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || authService.AuthUser(user, pass) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("The provided user is unauthorized"))
		return
	}
	switch r.Method {
	case "GET":
		data, err := ioutil.ReadFile(*statePath)
		if os.IsNotExist(err) {
			fmt.Fprintf(w, "")
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			return
		}
		w.Write(data)
		log.Println("State requested by", user)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			return
		}
		err = ioutil.WriteFile(*statePath, body, 0600)
		if err != nil {
			log.Println("Error writing state: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			return
		}
		log.Println("State Updated by", user)
	case "DELETE":
		err := os.Remove(*statePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			log.Println("Error removing state: ", err)
			return
		}
		log.Println("State purged by", user)
	default:
		log.Println("Unknown request verb")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("The only allowed methods are GET, POST, and DELETE"))
	}
}

func manageLocks(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || authService.AuthUser(user, pass) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("The provided user is unauthorized"))
		return
	}
	if r.Method != "LOCK" && r.Method != "UNLOCK" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("The only allowed methods are LOCK and UNLOCK"))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An internal error has occured"))
		log.Println("Error reading body:", err)
		return
	}
	info := &state.LockInfo{}
	err = json.Unmarshal(b, info)
	if err != nil && len(b) > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An internal error has occured"))
		log.Println("Error parsing lock:", err)
		log.Println(b)
		return
	}

	lockpath := fmt.Sprintf("%s.%s", *statePath, "lock")
	switch r.Method {
	case "LOCK":
		data, err := ioutil.ReadFile(lockpath)
		if err != nil && !os.IsNotExist(err) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			log.Println("Error reading lock:", err)
			return
		}
		if os.IsNotExist(err) {
			// No lock, write this one
			err := ioutil.WriteFile(lockpath, b, 0600)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("An internal error has occured"))
				log.Println("Error writing lock: ", err)
				return
			}
			log.Println("Lock issued to", user)
			fmt.Fprintf(w, "OK")
			return
		}
		// Made it this far?  There's a lock being held, fish
		// it out and send it back.
		w.WriteHeader(http.StatusConflict)
		w.Write(data)
		log.Println("A lock was denied to", user)
	case "UNLOCK":
		err := os.Remove(lockpath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			log.Println("Error clearing lock:", err)
			return
		}
		fmt.Fprintf(w, "OK")
		log.Println("Lock cleared by", user)
	}
}

func main() {
	flag.Parse()

	log.Println("Starting Terrastate")

	log.Println("The following authenticators are known")
	for _, b := range auth.List() {
		log.Printf("  %s", b)
	}

	var err error
	authService, err = auth.New()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/state", manageState)
	http.HandleFunc("/locks", manageLocks)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *addr, *port), nil))
}
