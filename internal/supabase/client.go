package supabase

import (
	"log"

	supabasego "github.com/supabase-community/supabase-go"
)

type Client struct {
	*supabasego.Client
}

func New(url, key string) *Client {
	c, err := supabasego.NewClient(url, key, nil)
	if err != nil {
		log.Fatal("Failed to initialize Supabase client:", err)
	}
	return &Client{Client: c}
}

func NewPublic(url, anonKey string) *Client {
	log.Println("Public Supabase client initialized")
	return New(url, anonKey)
}

func NewAdmin(url, serviceKey string) *Client {
	log.Println("Admin Supabase client initialized")
	return New(url, serviceKey)
}
