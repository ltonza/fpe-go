package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"encoding/hex"
	"regexp"

	"github.com/capitalone/fpe/ff1"
	"github.com/gorilla/mux"
	"gopkg.in/natefinch/lumberjack.v2"
)

func RequestLogger(targetMux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					start := time.Now()

					targetMux.ServeHTTP(w, r)

					log.Printf(
									"%s %s \t%v",
									r.Method,
									r.RequestURI,
									time.Since(start),
					)
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	original := ""
  alphabet := "0-9"
	switch r.Method {
	case "GET":     
  	query := r.URL.Query()
		log.Printf("query parameters = %v\n", query)
		original = query.Get("plain")
		alphabet = query.Get("alphabet")
  case "POST":
    // Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
    if err := r.ParseForm(); err != nil {
        fmt.Fprintf(w, "ParseForm() err: %v", err)
        return
		}
		log.Printf("form values = %v\n", r.PostForm)
		original = r.FormValue("plain")
		alphabet = r.FormValue("alphabet")
	}

	if original == "" {
		w.Write([]byte("Informe o texto..."))
		return 
	}

	// remove os caracteres inválidos do processo
	reg, _ := regexp.Compile("[^" + alphabet + "]+")
	changedString := reg.ReplaceAllString(original, "") 

	//determina de acordo com o alfabeto
	radix := 10
  switch alphabet {
	case "0-9":
		radix = 10   
	case "0-9a-z":
		radix = 36   
	case "0-9a-zA-Z":
		radix = 62   
	default:
		radix = 62   
	}

	// Key and tweak should be byte arrays. Put your key and tweak here.
	// To make it easier for demo purposes, decode from a hex string here.
	key, err := hex.DecodeString("EF4359D8D580AA4F7F036D6F04FC6A94")
	if err != nil {
		log.Fatal(err)
//		panic(err)
	}
	tweak, err := hex.DecodeString("D8E7920AFA330A73")
	if err != nil {
		log.Fatal(err)
//		panic(err)
	}

	// Create a new FF1 cipher "object"
	// 10 is the radix/base, and 8 is the tweak length.
	FF1, err := ff1.NewCipher(radix, 8, key, tweak)
	if err != nil {
		log.Fatal(err)
//		panic(err)
	}

	// Call the encryption function on an example SSN
	ciphertext, err := FF1.Encrypt(changedString)
	if err != nil {
		log.Fatal(err)
//		panic(err)
	}

	plaintext, err := FF1.Decrypt(ciphertext)
	if err != nil {
		log.Fatal(err)
//		panic(err)
  }

  if ( len(changedString) != len(original) ){
		for i, ch := range original {
			if ( rune(changedString[i]) != ch ){
				changedString = changedString[:i] + string(ch) + changedString[i:]
				ciphertext = ciphertext[:i] + string(ch) + ciphertext[i:]
				plaintext = plaintext[:i] + string(ch) + plaintext[i:]
			}
	  }
	}

	w.Write([]byte(fmt.Sprintf("Alphabet..: \t%s\n", alphabet)))
	w.Write([]byte(fmt.Sprintf("Original..: \t%s\n", original)))
	w.Write([]byte(fmt.Sprintf("Ciphered..: \t%s\n", ciphertext)))
	w.Write([]byte(fmt.Sprintf("Deciphered: \t%s\n", plaintext)))

}

func main() {
	// Create Server and Route Handlers
	r := mux.NewRouter()

	r.HandleFunc("/fpe", handler)

	r.PathPrefix("/").Handler(http.StripPrefix("/",http.FileServer(http.Dir("./static"))))

	port := os.Getenv("PORT") 
	if port == "" {
		 port = "8080"
	}

	srv := &http.Server{
		Handler:      RequestLogger(r),
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Configure Logging
	LOG_FILE_LOCATION := os.Getenv("LOG_FILE_LOCATION")
	if LOG_FILE_LOCATION != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   LOG_FILE_LOCATION,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   //days
			Compress:   true, // disabled by default
		})
	}

	// Start Server
	go func() {
		log.Println(`Starting Server: http:\\localhost:` + port)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
//			panic(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
