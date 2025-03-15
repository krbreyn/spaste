package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

func main() {
	server := SocketPasteServer{
		store: &MemoryPasteStore{
			endpoints: make(map[string]string),
			kg: KeyGenerator{
				taken: make(map[string]bool),
			},
		},
	}

	server.store.Set("google.com")
	for i, j := range server.store.endpoints {
		fmt.Println(i, j)
	}

	tcpPort := ":1337"
	listener, err := net.Listen("tcp", tcpPort)
	if err != nil {
		panic(err)
	}

	go server.acceptNetcats(listener)
	fmt.Println("listening to tcp on", tcpPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.ServeHTTP)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, //1MB
	}

	log.Printf("starting http server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type SocketPasteServer struct {
	store *MemoryPasteStore
}

func (s *SocketPasteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	paste := s.store.Get(path.Base(r.URL.Path))
	if paste != "" {
		_, _ = w.Write([]byte(paste))
		return
	}
	http.NotFound(w, r)
}

func (s *SocketPasteServer) acceptNetcats(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleNetcats(conn)
	}
}

const maxSize = 2 * 1024 * 1024 //2mb limit

func (s *SocketPasteServer) handleNetcats(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	var buf bytes.Buffer
	chunk := make([]byte, 1024*50) //50kb chunks

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			if buf.Len()+n > maxSize {
				_, _ = conn.Write([]byte("Input too large\n"))
				return
			}
			buf.Write(chunk[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}
	}
	paste := buf.String()

	if strings.TrimSpace(paste) == "" {
		_, _ = conn.Write([]byte("Don't send empty spaces!"))
		return
	}

	key := s.store.Set(paste)
	_, _ = conn.Write(fmt.Appendln(nil, "key is", key))
}

type MemoryPasteStore struct {
	endpoints map[string]string
	kg        KeyGenerator
	mu        sync.Mutex
}

func (store *MemoryPasteStore) Set(paste string) string {
	store.mu.Lock()
	defer store.mu.Unlock()

	key := store.kg.GenKey()
	store.endpoints[key] = paste
	return key
}

func (store *MemoryPasteStore) Get(key string) string {
	store.mu.Lock()
	defer store.mu.Unlock()

	if url, ok := store.endpoints[key]; ok {
		return url
	}
	return ""
}

type KeyGenerator struct {
	taken map[string]bool
}

const letters = "abcdefghijklmnopqrstuvwxyz123456789"

func (kg *KeyGenerator) GenKey() string {
	for {
		b := make([]byte, 6)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}

		key := string(b)
		if _, ok := kg.taken[key]; !ok {
			kg.taken[key] = true
			return key
		}
	}
}
