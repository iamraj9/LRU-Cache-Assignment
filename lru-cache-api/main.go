package main

import (
	"container/list"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// CacheItem to represents the cache item
type CacheItem struct {
	Key       string
	Value     interface{}
	ExpiresAt time.Time
}

// LRUCache implements
type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	list     *list.List
	mutex    sync.RWMutex
}

// NewLRUCache --- LRU cache with the given capacity
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves an item from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if element, exists := c.items[key]; exists {
		item := element.Value.(*CacheItem)
		if time.Now().After(item.ExpiresAt) {
			return nil, false
		}
		c.list.MoveToFront(element)
		return item.Value, true
	}
	return nil, false
}

// Set :: adding or updating an item in the cache
func (c *LRUCache) Set(key string, value interface{}, expiration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.items[key]; exists {
		c.list.MoveToFront(element)
		item := element.Value.(*CacheItem)
		item.Value = value
		item.ExpiresAt = time.Now().Add(expiration)
	} else {
		if c.list.Len() >= c.capacity {
			c.evict()
		}
		item := &CacheItem{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Now().Add(expiration),
		}
		element := c.list.PushFront(item)
		c.items[key] = element
	}
}

// Delete :: removes an item from the cache
func (c *LRUCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.items[key]; exists {
		c.list.Remove(element)
		delete(c.items, key)
	}
}

// evict :-> removes the least recently used item from the cache
func (c *LRUCache) evict() {
	if element := c.list.Back(); element != nil {
		item := element.Value.(*CacheItem)
		c.list.Remove(element)
		delete(c.items, item.Key)
	}
}

var (
	cache    *LRUCache
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in this example
		},
	}
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan CacheUpdate)
)

// CacheUpdate represents a cache update to be sent via WebSocket
type CacheUpdate struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expiresAt"`
}

func main() {
	cache = NewLRUCache(100) // Set cache capacity to 100 items

	r := mux.NewRouter()
	r.HandleFunc("/cache/{key}", getHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/cache/{key}", deleteHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/ws", handleWebSocket)
	r.HandleFunc("/cache", getAllCacheItems).Methods("GET")
	r.HandleFunc("/cache", setHandler).Methods("POST", "OPTIONS")

	go handleBroadcasts()
	go cleanupExpiredItems()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Wrap router with CORS and logging middleware
	handler := c.Handler(r)
	handler = logMiddleware(handler)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	value, found := cache.Get(key)
	if !found {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"key": key, "value": value})
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Key        string      `json:"key"`
		Value      interface{} `json:"value"`
		Expiration int         `json:"expiration"` // in seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expiration := time.Duration(data.Expiration) * time.Second
	cache.Set(data.Key, data.Value, expiration)

	broadcast <- CacheUpdate{
		Key:       data.Key,
		Value:     data.Value,
		ExpiresAt: time.Now().Add(expiration),
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Key set successfully"})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	cache.Delete(key)

	broadcast <- CacheUpdate{
		Key:       key,
		Value:     nil,
		ExpiresAt: time.Time{},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Key deleted successfully"})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	// Send current cache state to the new client
	cache.mutex.RLock()
	for _, element := range cache.items {
		item := element.Value.(*CacheItem)
		update := CacheUpdate{
			Key:       item.Key,
			Value:     item.Value,
			ExpiresAt: item.ExpiresAt,
		}
		err := conn.WriteJSON(update)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, conn)
			return
		}
	}
	cache.mutex.RUnlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, conn)
			break
		}
	}
}

func handleBroadcasts() {
	for update := range broadcast {
		for client := range clients {
			err := client.WriteJSON(update)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func cleanupExpiredItems() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cache.mutex.Lock()
		for key, element := range cache.items {
			item := element.Value.(*CacheItem)
			if time.Now().After(item.ExpiresAt) {
				cache.list.Remove(element)
				delete(cache.items, key)
				broadcast <- CacheUpdate{
					Key:       key,
					Value:     nil,
					ExpiresAt: time.Time{},
				}
			}
		}
		cache.mutex.Unlock()
	}
}

func getAllCacheItems(w http.ResponseWriter, r *http.Request) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	items := make(map[string]interface{})
	for key, element := range cache.items {
		item := element.Value.(*CacheItem)
		if time.Now().Before(item.ExpiresAt) {
			items[key] = map[string]interface{}{
				"value":     item.Value,
				"expiresAt": item.ExpiresAt,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
