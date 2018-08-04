package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/hashicorp/terraform/state"
)

var (
	statePath = flag.String("state_file", "./state.dat", "Location for the state file")
)

func manageState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		log.Println("Dispensing State")
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
	case "POST":
		log.Println("Updating State")
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
		log.Println("State Updated")
	case "DELETE":
		log.Println("Purging State")
		err := os.Remove(*statePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			log.Println("Error removing state: ", err)
			return
		}
		log.Println("State purged")
	default:
		log.Println("Unknown request verb")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("The only allowed methods are GET, POST, and DELETE"))
	}
}

func manageLocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "LOCK" && r.Method != "UNLOCK" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("The only allowed methods are LOCK and UNLOCK"))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An internal error has occured"))
		log.Println("Error reading body: ", err)
		return
	}
	info := &state.LockInfo{}
	err = json.Unmarshal(b, info)
	if err != nil && len(b) > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An internal error has occured"))
		log.Println("Error parsing lock: ", err)
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
			log.Println("Error reading lock: ", err)
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
			log.Println("Lock issued")
			fmt.Fprintf(w, "OK")
			return
		}
		// Made it this far?  There's a lock being held, fish
		// it out and send it back.
		w.WriteHeader(http.StatusConflict)
		w.Write(data)
	case "UNLOCK":
		err := os.Remove(lockpath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("An internal error has occured"))
			log.Println("Error clearing lock: ", err)
			return
		}
		fmt.Fprintf(w, "OK")
		log.Println("Lock cleared")
	}
}

func main() {
	log.Println("Starting Terrastate")
	http.HandleFunc("/state", manageState)
	http.HandleFunc("/locks", manageLocks)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
