package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

// A CacheEntry represents an answer to a client
type CacheEntry struct {
	Status    int
	Body      []byte
	Requested time.Time
	Headers   map[string][]string
}

var cache map[string]CacheEntry

func replaceBaseURL(content []byte) []byte {

	re := regexp.MustCompile(os.Getenv("REWRITE_FROM"))
	return []byte(re.ReplaceAllString(string(content), os.Getenv("REWRITE_TO")))
}

func updateCache(method string, path string, donechan chan<- CacheEntry, errchan chan<- error) {
	log.Printf("Updating cache: %v\n", path)
	client := &http.Client{}
	req, err := http.NewRequest(method, os.Getenv("PROXY_BASE_URL")+path, nil)
	if err != nil {
		errchan <- err
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		errchan <- err
		return
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errchan <- err
		return
	}

	respBytes = replaceBaseURL(respBytes)

	log.Printf("Updated cache: %v\n", path)

	headerMap := make(map[string][]string)
	for key, val := range resp.Header {
		headerMap[key] = val
	}

	donechan <- CacheEntry{
		Status:    resp.StatusCode,
		Body:      respBytes,
		Requested: time.Now(),
		Headers:   headerMap,
	}
}

func serveCacheEntry(w http.ResponseWriter, cacheentry CacheEntry) {
	for key, values := range cacheentry.Headers {
		if key != "Content-Length" {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}
	}
	w.WriteHeader(cacheentry.Status)
	w.Write(cacheentry.Body)
}

func main() {
	cache = make(map[string]CacheEntry)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		done := make(chan CacheEntry)
		err := make(chan error)

		if cacheentry, ok := cache[r.URL.Path]; ok {
			go updateCache(r.Method, r.URL.Path, done, err)
			serveCacheEntry(w, cacheentry)
			log.Printf("Served old content from cache: %v\n", r.URL.Path)
			return
		}

		go updateCache(r.Method, r.URL.Path, done, err)
		select {
		case cacheentry := <-done:
			cache[r.URL.Path] = cacheentry
			serveCacheEntry(w, cacheentry)
			log.Printf("Served fresh content from cache: %v\n", r.URL.Path)
			return
		case <-err:
			w.WriteHeader(500)
		}
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
