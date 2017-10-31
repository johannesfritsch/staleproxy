package main

import (
	"fmt"
	"io"
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

func getCacheKey(method string, path string, query string) *string {
	var key *string
	if method == "GET" {
		newkey := fmt.Sprintf("%v#%v#%v", method, path, query)
		key = &newkey
	} else {
		key = nil
	}
	return key
}

func getRequestPath(path string, query string) string {
	if query == "" {
		return os.Getenv("PROXY_BASE_URL") + path
	} else {
		return os.Getenv("PROXY_BASE_URL") + path + "?" + query
	}
}

func updateCache(method string, path string, query string, body io.Reader, donechan chan<- CacheEntry, errchan chan<- error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, getRequestPath(path, query), body)
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

	headerMap := make(map[string][]string)
	for key, val := range resp.Header {
		headerMap[key] = val
	}

	cacheentry := CacheEntry{
		Status:    resp.StatusCode,
		Body:      respBytes,
		Requested: time.Now(),
		Headers:   headerMap,
	}

	if cachekey := getCacheKey(method, path, query); cachekey != nil {
		log.Printf("lookup cache key for update: %v\n", *cachekey)
		cache[*cachekey] = cacheentry
	}

	donechan <- cacheentry
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

		if cachekey := getCacheKey(r.Method, r.URL.Path, r.URL.RawQuery); cachekey != nil {
			log.Printf("Lookup cache key: %v\n", *cachekey)
			if cacheentry, ok := cache[*cachekey]; ok {
				go updateCache(r.Method, r.URL.Path, r.URL.RawQuery, r.Body, done, err)
				serveCacheEntry(w, cacheentry)
				log.Printf("Served old content from cache: %v\n", r.URL.Path)
				return
			}
		}

		go updateCache(r.Method, r.URL.Path, r.URL.RawQuery, r.Body, done, err)
		select {
		case cacheentry := <-done:
			serveCacheEntry(w, cacheentry)
			log.Printf("Served fresh content from cache: %v\n", r.URL.Path)
			return
		case <-err:
			w.WriteHeader(500)
		}
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
