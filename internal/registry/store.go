package registry

import (
	"encoding/json"
	"fmt"
	"os"
)

type Store struct {
	filePath string
}

func NewStore(path string) *Store {
	return &Store{filePath: path}
}

func (s *Store) Save(client *Client) error {
	clients, _ := s.List()

	// Check if client exists
	for i, c := range clients {
		if c.ID == client.ID {
			clients[i] = client
			return s.writeAll(clients)
		}
	}

	// Add new client
	clients = append(clients, client)
	return s.writeAll(clients)
}

func (s *Store) List() ([]*Client, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Client{}, nil
		}
		return nil, err
	}

	var clients []*Client
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, err
	}

	return clients, nil
}

func (s *Store) Get(id string) (*Client, error) {
	clients, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, c := range clients {
		if c.ID == id {
			return c, nil
		}
	}

	return nil, fmt.Errorf("client %s not found", id)
}

func (s *Store) Delete(id string) error {
	clients, err := s.List()
	if err != nil {
		return err
	}

	newClients := []*Client{}
	for _, c := range clients {
		if c.ID != id {
			newClients = append(newClients, c)
		}
	}

	return s.writeAll(newClients)
}

func (s *Store) writeAll(clients []*Client) error {
	data, err := json.MarshalIndent(clients, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}
